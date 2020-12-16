package git

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	ghttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

type Options struct {
	Credential        *corev1.Secret
	CABundle          []byte
	InsecureTLSVerify bool
	Headers           map[string]string
}

func NewGit(directory, url string, opts *Options) (*Git, error) {
	if opts == nil {
		opts = &Options{}
	}

	g := &Git{
		URL:               url,
		Directory:         directory,
		caBundle:          opts.CABundle,
		insecureTLSVerify: opts.InsecureTLSVerify,
		secret:            opts.Credential,
		headers:           opts.Headers,
	}
	return g, g.setCredential(opts.Credential)
}

type Git struct {
	URL               string
	Directory         string
	user              string
	password          string
	caBundle          []byte
	insecureTLSVerify bool
	secret            *corev1.Secret
	headers           map[string]string
	knownHosts        []byte
	privateKey        []byte
	auth              transport.AuthMethod
}

// LsRemote runs ls-remote on git repo and returns the HEAD commit SHA
func (g *Git) LsRemote(branch string, commit string) (string, error) {
	if changed, err := g.remoteSHAChanged(branch, commit); err != nil || !changed {
		return commit, err
	}

	rem := gogit.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{g.URL},
	})

	refs, err := rem.List(&gogit.ListOptions{
		Auth:            g.auth,
		InsecureSkipTLS: g.insecureTLSVerify,
		CABundle:        g.caBundle,
	})
	if err != nil {
		return "", err
	}

	for _, ref := range refs {
		if ref.Name().String() == fmt.Sprintf("refs/heads/%s", branch) {
			return ref.Hash().String(), nil
		}
	}

	return "", fmt.Errorf("no commit for branch: %s, url: %s", branch, g.URL)
}

// Head runs git clone on directory(if not exist), reset dirty content and return the HEAD commit
func (g *Git) Head(branch string) (string, error) {
	repo, err := g.clone(branch)
	if err != nil {
		return "", err
	}

	if err := g.reset(repo, "HEAD"); err != nil {
		return "", err
	}

	return g.currentCommit(repo)
}

// Clone runs git clone with depth 1
func (g *Git) Clone(branch string) (*gogit.Repository, error) {
	return gogit.PlainClone(g.Directory, false, &gogit.CloneOptions{
		URL:           g.URL,
		SingleBranch:  true,
		NoCheckout:    true,
		Depth:         1,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		Progress:      os.Stdout,
	})
}

// Update updates git repo if remote sha has changed
func (g *Git) Update(branch string) (string, error) {
	repo, err := g.clone(branch)
	if err != nil {
		return "", nil
	}

	if err := g.reset(repo, "HEAD"); err != nil {
		return "", err
	}

	commit, err := g.currentCommit(repo)
	if err != nil {
		return commit, err
	}

	if changed, err := g.remoteSHAChanged(branch, commit); err != nil || !changed {
		return commit, err
	}

	if err := g.fetchAndCheckout(repo, branch, commit); err != nil {
		return "", err
	}

	return g.currentCommit(repo)
}

// Ensure runs git clone, clean DIRTY contents and fetch the latest commit
func (g *Git) Ensure(branch, commit string) error {
	repo, err := g.clone(branch)
	if err != nil {
		return err
	}

	if err := g.reset(repo, commit); err != nil {
		return nil
	}

	return g.fetchAndCheckout(repo, branch, commit)
}

func (g *Git) httpClientWithCreds() (*http.Client, error) {
	var (
		username  string
		password  string
		tlsConfig tls.Config
	)

	if g.secret != nil {
		switch g.secret.Type {
		case corev1.SecretTypeBasicAuth:
			username = string(g.secret.Data[corev1.BasicAuthUsernameKey])
			password = string(g.secret.Data[corev1.BasicAuthPasswordKey])
		case corev1.SecretTypeTLS:
			cert, err := tls.X509KeyPair(g.secret.Data[corev1.TLSCertKey], g.secret.Data[corev1.TLSPrivateKeyKey])
			if err != nil {
				return nil, err
			}
			tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
		}
	}

	if len(g.caBundle) > 0 {
		cert, err := x509.ParseCertificate(g.caBundle)
		if err != nil {
			return nil, err
		}
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		pool.AddCert(cert)
		tlsConfig.RootCAs = pool
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tlsConfig
	transport.TLSClientConfig.InsecureSkipVerify = g.insecureTLSVerify

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	if username != "" || password != "" {
		client.Transport = &basicRoundTripper{
			username: username,
			password: password,
			next:     client.Transport,
		}
	}

	return client, nil
}

func (g *Git) remoteSHAChanged(branch, sha string) (bool, error) {
	formattedURL := formatGitURL(g.URL, branch)
	if formattedURL == "" {
		return true, nil
	}

	client, err := g.httpClientWithCreds()
	if err != nil {
		logrus.Warnf("Problem creating http client to check git remote sha of repo [%v]: %v", g.URL, err)
		return true, nil
	}
	defer client.CloseIdleConnections()

	req, err := http.NewRequest("GET", formattedURL, nil)
	if err != nil {
		logrus.Warnf("Problem creating request to check git remote sha of repo [%v]: %v", g.URL, err)
		return true, nil
	}

	req.Header.Set("Accept", "application/vnd.github.v3.sha")
	req.Header.Set("If-None-Match", fmt.Sprintf("\"%s\"", sha))
	for k, v := range g.headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Return timeout errors so caller can decide whether or not to proceed with updating the repo
		uErr := &url.Error{}
		if ok := errors.As(err, &uErr); ok && uErr.Timeout() {
			return false, errors.Wrapf(uErr, "Repo [%v] is not accessible", g.URL)
		}
		return true, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return false, nil
	}

	return true, nil
}

func (g *Git) setCredential(cred *corev1.Secret) error {
	if cred == nil {
		return nil
	}

	if cred.Type == corev1.SecretTypeBasicAuth {
		username, password := cred.Data[corev1.BasicAuthUsernameKey], cred.Data[corev1.BasicAuthPasswordKey]
		if len(password) == 0 && len(username) == 0 {
			return nil
		}

		u, err := url.Parse(g.URL)
		if err != nil {
			return err
		}
		u.User = url.User(string(username))
		g.user = string(username)
		g.URL = u.String()
		g.password = string(password)
	} else if cred.Type == corev1.SecretTypeSSHAuth {
		g.privateKey = cred.Data[corev1.SSHAuthPrivateKey]
		g.knownHosts = cred.Data["known_hosts"]
	}

	var err error
	g.auth, err = g.setupAuth()
	return err
}

func (g *Git) clone(branch string) (*gogit.Repository, error) {
	gitDir := filepath.Join(g.Directory, ".git")
	if dir, err := os.Stat(gitDir); err == nil && dir.IsDir() {
		return gogit.PlainOpen(g.Directory)
	}

	if err := os.RemoveAll(g.Directory); err != nil {
		return nil, fmt.Errorf("failed to remove directory %s: %v", g.Directory, err)
	}

	return gogit.PlainClone(g.Directory, false, &gogit.CloneOptions{
		URL:             g.URL,
		Auth:            g.auth,
		SingleBranch:    true,
		NoCheckout:      true,
		Depth:           1,
		ReferenceName:   plumbing.NewBranchReferenceName(branch),
		Progress:        os.Stdout,
		InsecureSkipTLS: g.insecureTLSVerify,
		CABundle:        g.caBundle,
	})
}

func (g *Git) fetchAndCheckout(repo *gogit.Repository, branch, commit string) error {
	if err := repo.Fetch(&gogit.FetchOptions{
		RemoteName: "origin",
		Auth:       g.auth,
		RefSpecs: []config.RefSpec{
			refName(branch),
		},
		InsecureSkipTLS: g.insecureTLSVerify,
		CABundle:        g.caBundle,
		Progress:        os.Stdout,
	}); err != nil && err != gogit.NoErrAlreadyUpToDate {
		return err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	if err := wt.Reset(&gogit.ResetOptions{
		Mode: gogit.HardReset,
	}); err != nil {
		return err
	}
	return wt.Checkout(&gogit.CheckoutOptions{
		Hash: plumbing.NewHash(commit),
	})
}

func (g *Git) reset(repo *gogit.Repository, rev string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	return wt.Reset(&gogit.ResetOptions{
		Commit: plumbing.NewHash(rev),
		Mode:   gogit.HardReset,
	})
}

func (g *Git) currentCommit(repo *gogit.Repository) (string, error) {
	head, err := repo.Head()
	return head.Hash().String(), err
}

func (g *Git) setupAuth() (transport.AuthMethod, error) {
	if g.user != "" || g.password != "" {
		return &ghttp.BasicAuth{
			Username: g.user,
			Password: g.password,
		}, nil
	}

	if len(g.privateKey) > 0 {
		publicKey, err := ssh.NewPublicKeys("git", g.privateKey, "")
		if err != nil {
			return nil, err
		}
		if len(g.knownHosts) > 0 {
			f, err := ioutil.TempFile("", "known_hosts")
			if err != nil {
				return nil, err
			}
			defer os.RemoveAll(f.Name())
			defer f.Close()

			if _, err := f.Write(g.knownHosts); err != nil {
				return nil, err
			}
			if err := f.Close(); err != nil {
				return nil, fmt.Errorf("closing knownHosts file %s: %w", f.Name(), err)
			}
			callback, err := ssh.NewKnownHostsCallback(f.Name())
			if err != nil {
				return nil, err
			}
			publicKey.HostKeyCallback = callback
		}

		return publicKey, nil
	}
	return nil, nil
}

func formatGitURL(endpoint, branch string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}

	pathParts := strings.Split(u.Path, "/")
	switch u.Hostname() {
	case "github.com":
		if len(pathParts) >= 3 {
			org := pathParts[1]
			repo := strings.TrimSuffix(pathParts[2], ".git")
			return fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", org, repo, branch)
		}
	case "git.rancher.io":
		repo := strings.TrimSuffix(pathParts[1], ".git")
		u.Path = fmt.Sprintf("/repos/%s/commits/%s", repo, branch)
		return u.String()
	}

	return ""
}

type basicRoundTripper struct {
	username string
	password string
	next     http.RoundTripper
}

func (b *basicRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request.SetBasicAuth(b.username, b.password)
	return b.next.RoundTrip(request)
}

func refName(ref string) config.RefSpec {
	return config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", ref, ref))
}
