package whatapmongo

type MongoClientOptions interface {
	GetURI() string
}
