package config

type Route struct {
	Path   string
	Target string
}

var Routes = []Route{
	{
		Path:   "/users",
		Target: "http://localhost:3001",
	},
	{
		Path:   "/orders",
		Target: "http://localhost:3002",
	},
}
