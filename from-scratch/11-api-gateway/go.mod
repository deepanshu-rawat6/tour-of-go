module tour_of_go/projects/from-scratch/11-api-gateway

go 1.25

replace tour_of_go/projects/from-scratch/04-rate-limiter => ../04-rate-limiter

require gopkg.in/yaml.v3 v3.0.1

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	tour_of_go/projects/from-scratch/04-rate-limiter v0.0.0-00010101000000-000000000000
)

require github.com/go-chi/chi/v5 v5.2.1
