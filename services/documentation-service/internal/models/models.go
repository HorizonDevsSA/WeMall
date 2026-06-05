package models

type ModelField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type DataModel struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Fields      []ModelField `json:"fields"`
}

type Endpoint struct {
	Name         string   `json:"name"`
	Protocol     string   `json:"protocol"` // "GraphQL Query" | "GraphQL Mutation" | "gRPC" | "HTTP Route"
	Description  string   `json:"description"`
	AuthRequired bool     `json:"auth_required"`
	Roles        []string `json:"roles"`
	RequestBody  string   `json:"request_body"`
	ResponseBody string   `json:"response_body"`
}

type APICategory struct {
	Slug        string      `json:"slug"`
	Title       string      `json:"title"`
	Overview    string      `json:"overview"`
	Icon        string      `json:"icon"`
	Endpoints   []Endpoint  `json:"endpoints"`
	DataModels  []DataModel `json:"data_models"`
}

type SystemStats struct {
	TotalCategories int
	TotalEndpoints  int
	Microservices   int
	Protocols       int
	GatewayPort     string
	Version         string
}
