Test the services with this endpoint

The rest endpoint for unbound grpc service methods is form as described here
https://github.com/grpc-ecosystem/grpc-gateway/blob/master/docs/docs/mapping/grpc_api_configuration.md#generate_unbound_methods 

For example

curl -XPOST http://localhost:8080/hello.Hello/hello  -d '{"Name": " Vincent", "Greeting": "Howdy"}'^C