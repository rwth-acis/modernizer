import weaviate
import os
import requests
import json
import os
import weaviate.classes as wvc

# Connect to a local Weaviate instance
client = weaviate.connect_to_custom(
    http_host="192.168.10.165",
    http_port=8080,
    http_secure=False,
    grpc_host="192.168.10.165",
    grpc_port=50051,
    grpc_secure=False,
    auth_credentials=weaviate.auth.AuthApiKey(
        os.getenv("WEAVIATE_KEY")
    ),  # Set this environment variable
)

try:
    # Wrap in try/finally to ensure client is closed gracefully
    # ===== define collection =====
    questions = client.collections.create(
        name="JeopardyCategory",
    )

    client.collections.create(
        name="JeopardyQuestion",
        description="A Jeopardy! question",
        properties=[
            wvc.config.Property(name="question", data_type=wvc.config.DataType.TEXT),
            wvc.config.Property(name="answer", data_type=wvc.config.DataType.TEXT),
        ],
        references=[
            wvc.config.ReferenceProperty(
                name="hasCategory",
                target_collection="JeopardyCategory"
            )
        ]

    )


finally:
    client.close()  # Close client gracefully
