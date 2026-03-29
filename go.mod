module github.com/LittleAksMax/bids-service

go 1.25

require (
	github.com/allegro/bigcache/v3 v3.1.0
	github.com/LittleAksMax/bids-util v0.0.6-0.20260325145548-ce2ed8e49d4c
	github.com/google/uuid v1.6.0
)

replace github.com/LittleAksMax/amazon-ads-api-sdk-go => ../amazon-ads-api-go-sdk

require (
	github.com/LittleAksMax/amazon-ads-api-sdk-go v0.0.0-00010101000000-000000000000
	github.com/LittleAksMax/bidscript v0.0.0-20260326213106-da8fa7158626
	github.com/joho/godotenv v1.5.1
	github.com/rabbitmq/amqp091-go v1.10.0
)
