module github.com/olegromanchuk/hotelito

go 1.19

replace github.com/olegromanchuk/hotelito/pkg/hotel => ./pkg/hotel

replace github.com/olegromanchuk/hotelito/pkg/pbx3cx => ./pkg/pbx/pbx3cx

replace github.com/olegromanchuk/hotelito/pkg/pbx => ./pkg/pbx

replace github.com/olegromanchuk/hotelito/pkg/secrets => ./pkg/secrets

require (
	github.com/gorilla/mux v1.8.0
	github.com/joho/godotenv v1.5.1
	github.com/sirupsen/logrus v1.9.3
	go.etcd.io/bbolt v1.3.7
	golang.org/x/oauth2 v0.9.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.11.0 // indirect
	golang.org/x/sys v0.9.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)
