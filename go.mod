module github.com/mdblp/shoreline

go 1.12

replace github.com/tidepool-org/shoreline => ./

replace github.com/tidepool-org/go-common => github.com/mdblp/go-common v0.3.0

require (
	github.com/RangelReale/osin v1.0.1
	github.com/SpeakData/minimarketo v0.0.0-20170821092521-29339e452f44
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/codegangsta/cli v1.20.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/gorilla/mux v1.7.3
	github.com/prometheus/client_golang v1.4.1
	github.com/swaggo/swag v1.6.5
	github.com/tidepool-org/go-common v0.0.0-00010101000000-000000000000
	github.com/tidepool-org/shoreline v0.0.0-00010101000000-000000000000
)
