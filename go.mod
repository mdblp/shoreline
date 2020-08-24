module github.com/mdblp/shoreline

go 1.14

replace github.com/tidepool-org/shoreline => ./

replace github.com/tidepool-org/go-common => ../go-common

require (
	github.com/SpeakData/minimarketo v0.0.0-20170821092521-29339e452f44
	github.com/urfave/cli/v2 v2.2.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/gorilla/mux v1.7.4
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/prometheus/client_golang v1.7.1
	github.com/tidepool-org/go-common v0.0.0-00010101000000-000000000000
	github.com/tidepool-org/shoreline v0.0.0-00010101000000-000000000000
)
