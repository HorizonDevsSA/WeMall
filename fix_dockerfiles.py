import os
import glob
import re

directories = [
    "pkg",
    "gen",
    "services/api-gateway",
    "services/documentation-service",
    "services/media-service",
    "services/notification-service",
    "services/order-service",
    "services/product-service",
    "services/review-service",
    "services/seller-service",
    "services/user-service",
    "services/payment-service",
    "services/chat-service",
    "services/dispute-service",
    "services/admin-service",
    "services/promotion-service",
    "services/recommendation-service"
]

copy_block = ""
for d in directories:
    copy_block += f"COPY {d}/go.mod ./{d}/\n"

for dockerfile in glob.glob("services/*/Dockerfile"):
    with open(dockerfile, 'r') as f:
        content = f.read()
    
    # Replace the block of COPY .../go.mod with our new block
    # We find where `COPY pkg/go.mod` starts, and replace everything up to `RUN go work sync`
    new_content = re.sub(
        r'COPY pkg/go\.mod.*?(?=RUN go work sync)',
        copy_block + '\n',
        content,
        flags=re.DOTALL
    )
    
    with open(dockerfile, 'w') as f:
        f.write(new_content)
    print(f"Updated {dockerfile}")

