import json
import urllib.request
import urllib.error

GATEWAY_URL = "https://15.240.45.232.nip.io/graphql"

def gql(query, variables=None):
    headers = {"Content-Type": "application/json"}
    payload = {"query": query}
    if variables:
        payload["variables"] = variables
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(GATEWAY_URL, data=data, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            res = json.loads(response.read().decode("utf-8"))
            if "errors" in res:
                print(f"GraphQL error: {res['errors']}")
                return None
            return res.get("data")
    except Exception as e:
        print(f"Exception: {e}")
        return None

def print_tree(categories, indent=0):
    for c in categories:
        print("  " * indent + f"- {c['name']} (ID: {c['id']}, Slug: {c['slug']})")
        if "children" in c and c["children"]:
            print_tree(c["children"], indent + 1)

def main():
    query = """
    query {
      categories {
        id
        name
        slug
        children {
          id
          name
          slug
          children {
            id
            name
            slug
          }
        }
      }
    }
    """
    res = gql(query)
    if res and res.get("categories"):
        print_tree(res["categories"])
    else:
        print("No categories found.")

if __name__ == "__main__":
    main()
