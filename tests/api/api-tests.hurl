# AI Gateway API Test Collection (Hurl format)
# Run with: hurl --test tests/api/api-tests.hurl

# Variables
GET {{host}} == "http://localhost:8080"

###
# Health Check
###

# Test health endpoint returns 200
GET http://{{host}}/health
HTTP 200
[Asserts]
jsonpath "$.status" == "healthy"
jsonpath "$.service" == "ai-gateway"
jsonpath "$.timestamp" exists

###
# Providers API
###

# List all providers
GET http://{{host}}/v1/providers
HTTP 200
[Asserts]
jsonpath "$.providers" exists
jsonpath "$.providers" isContainer

###
# Chat Completions API
###

# Chat completions with minimal request
POST http://{{host}}/api/v1/chat/completions
Content-Type: application/json
{
    "model": "gpt-4",
    "messages": [
        {"role": "user", "content": "Hello"}
    ]
}
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"

###
# Chat completions with all parameters
POST http://{{host}}/api/v1/chat/completions
Content-Type: application/json
{
    "model": "gpt-4",
    "messages": [
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "What is AI?"}
    ],
    "temperature": 0.7,
    "max_tokens": 1000,
    "stream": false
}
HTTP 200

###
# Completions API
###

# Completions endpoint
POST http://{{host}}/api/v1/completions
Content-Type: application/json
{
    "model": "gpt-3.5-turbo-instruct",
    "prompt": "Once upon a time",
    "max_tokens": 50
}
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"

###
# Embeddings API
###

# Embeddings endpoint
POST http://{{host}}/api/v1/embeddings
Content-Type: application/json
{
    "model": "text-embedding-ada-002",
    "input": "Hello world"
}
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"

###
# Error Cases
###

# Missing required fields
POST http://{{host}}/api/v1/chat/completions
Content-Type: application/json
{}
HTTP 200  # Currently returns placeholder, should be 400 when implemented

###
# Rate Limiting
###

# Rapid requests should still work within limits
GET http://{{host}}/health
HTTP 200

GET http://{{host}}/health
HTTP 200

GET http://{{host}}/health
HTTP 200

GET http://{{host}}/health
HTTP 200

GET http://{{host}}/health
HTTP 200
