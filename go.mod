module github.com/LittleAksMax/bids-service

go 1.25.8

require (
	github.com/LittleAksMax/bids-util v0.0.6-0.20260325145548-ce2ed8e49d4c
	github.com/allegro/bigcache/v3 v3.1.0
	github.com/google/uuid v1.6.0
)

replace github.com/LittleAksMax/amazon-ads-api-sdk-go => ../amazon-ads-api-go-sdk
replace github.com/LittleAksMax/bids-util => ../bids-util

require (
	github.com/LittleAksMax/amazon-ads-api-sdk-go v0.0.0-00010101000000-000000000000
	github.com/LittleAksMax/bidscript v0.0.0-20260329160519-59e27fdefbc9
	github.com/joho/godotenv v1.5.1
)
