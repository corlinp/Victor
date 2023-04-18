# Victor

> "What's our vector, Victor?"

Victor is a dead-simple Vector database written in Go, optimized for the 1536-length vectors returned by the OpenAI embeddings API.

Victor is for people who want to play around with using vector stores for LLM memory and want something slightly better than an array of JSON objects,but don't want the complexity of standing up a real vector DB like Faiss, Milvus etc. Victor is a toy and NOT a production-ready database.

Victor uses Badger to store data and vectors on disk in an efficient protobuf format. The index is an in-memory B-tree built quickly from disk on startup. Victor uses dot products to find similarity, and each search scans the entire index - no fancy heuristic or clustering algorithms here - but it should be fast enough for your hobby project.

## Usage

```
go install github.com/corlinp/victor@latest
victor --data-dir /tmp/victor --host localhost:6723
```

Or use Docker:

```
docker run -p 6723:6723 -v /tmp/victor:/etc/victor-data corlinp/victor
```

## Real-world performance on my laptop

For a database with 100K vectors:
- Loading all vectors from a Python program via /add: `230 sec`
- Restoring index from disk (on startup): `1.5 sec`
- Memory use: `2.8 GB`
- Avg. call to /search: `0.2 sec`

If you're storing more than 100K vectors, you should probably consider using something else.


## API

`PUT /add`

```
{
    "id": "1",
    "vector": [1, 2, 3, ..., 1536],
    "data": "Hello, World!",
}
```

`POST /search`

```
{
    "vector": [1, 2, 3, ..., 1536],
    "count": 10,
}
```

returns
```
[
    {
        "id": "1",
        "distance": 0,
        "data": "Hello, World!",
    },
]
```

`GET /get/1`

```
Hello, World!
```

`DELETE /delete/1`

## Python Client

Here's a class to interact with Victor in Python!

```python
import requests
import json

class Victor:
    def __init__(self, base_url):
        self.base_url = base_url

    def add(self, id, vector, data=None):
        url = f'{self.base_url}/add'
        payload = {
            'id': id,
            'vector': vector,
            'data': data
        }
        response = requests.put(url, json=payload)
        response.raise_for_status()

    def search(self, vector, num_results):
        url = f'{self.base_url}/search'
        payload = {
            'vector': vector,
            'count': num_results
        }
        headers = {'Content-Type': 'application/json'}
        response = requests.post(url, data=json.dumps(payload), headers=headers)
        response.raise_for_status()
        return response.json()

    def get(self, id):
        url = f'{self.base_url}/get/{id}'
        response = requests.get(url)
        response.raise_for_status()
        return response.text

    def delete(self, id):
        url = f'{self.base_url}/delete/{id}'
        response = requests.delete(url)
        response.raise_for_status()
```