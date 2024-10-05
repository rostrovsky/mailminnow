#!/usr/bin/env bash

swaks --to recipient@example.com \
    --from sender@example.com \
    --server localhost:1025  \
    --header "Subject: Welcome to Our Newsletter" \
    --header "Content-Type: text/html" \
    --body "$(cat example-mail.html)"

swaks --to recipient@example.com \
    --from sender@example.com \
    --server localhost:1025 \
    --header "Subject: Test email" \
    --body "This is a test email sent using swaks."
