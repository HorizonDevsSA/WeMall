import json
import urllib.request
import urllib.error

GATEWAY_URL = "https://15.240.45.232.nip.io/graphql"

def gql(query, variables=None, token=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
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

def check_seller(email):
    print(f"\nChecking seller: {email}")
    auth = gql(f"""
        mutation {{
          sellerFirebaseSignIn(
            idToken: "mock-firebase-token-{email}",
            fullName: "Horizon Devs"
          ) {{
            accessToken
            user {{ id email fullName }}
          }}
        }}
    """)
    if not auth or not auth.get("sellerFirebaseSignIn"):
        print("  ✗ Authentication failed")
        return
    token = auth["sellerFirebaseSignIn"]["accessToken"]
    user = auth["sellerFirebaseSignIn"]["user"]
    print(f"  ✓ Authenticated: {user}")
    
    store_data = gql("query { myStore { id storeName status } }", token=token)
    if store_data and store_data.get("myStore"):
        print(f"  ✓ Store: {store_data['myStore']}")
    else:
        print("  ✗ No store found")

def main():
    check_seller("horizondevs19@gmail.com")
    check_seller("akotoxmpimbo@gmail.com")

if __name__ == "__main__":
    main()
