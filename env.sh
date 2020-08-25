export GATEKEEPER_SERVICE="http://localhost:9123"

# MONGODB_HOST="mongodb://localhost/"
# MONGODB_HOST="mongodb://personal:password@localhost:27017/user?authSource=admin&ssl=false"
export TIDEPOOL_SHORELINE_SERVICE='{
    "service": {
        "service": "user-api-local",
        "protocol": "http",
        "host": "localhost:9107",
        "keyFile": "config/key.pem",
        "certFile": "config/cert.pem"
    },
    "mongo": {
        "connectionString": "mongodb://localhost/"
    },
    "user": {
        "secrets": [{"secret": "default", "pass": "xxxxxxxxx"}, {"secret": "product_website", "pass": "xxxxxxxxx"}],
        "apiSecret": "This is a local API secret for everyone. BsscSHqSHiwrBMJsEGqbvXiuIUPAjQXU",
        "longTermKey": "abcdefghijklmnopqrstuvwxyz",
        "longTermDaysDuration": 30,
        "tokenDurationSecs": 2592000,
        "salt": "ADihSEI7tOQQP9xfXMO9HfRpXKu1NpIJ",
        "maxFailedLogin": 5,
        "delayBeforeNextLoginAttempt": 10,
        "maxConcurrentLogin": 100,
        "verificationSecret": "+skip",
        "clinicDemoUserId": ""

    }
}'
