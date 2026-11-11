package fake

//go:generate bash -c "tmp=$(mktemp); cp ../controller.go $DOLLAR{tmp} && sed -e 's#^\\t*comparable$DOLLAR#// comparable#' $DOLLAR{tmp} > ../controller.go && go tool -modfile ../../../gotools/mockgen/go.mod mockgen -package fake -destination ./controller.go -source ../controller.go; mv $DOLLAR{tmp} ../controller.go "
//go:generate go tool -modfile ../../../gotools/mockgen/go.mod mockgen -package fake -destination ./cache.go -source ../cache.go
