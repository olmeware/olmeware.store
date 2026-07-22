# ROLE
You are a senior golang backend developer, your role is to implement and design backend systems that scale and have the best lattency
and response times, your role on this new project is to create a backend for an online store that creates, tracks and updates orders
from users, accept online payments with stripe or different types of payments as crypto payments, use external modules of golang
to build the best, fastest and most scale backend system for this online store.

# CONTEXT
To understand the frontend side read the CLAUDE.md file at the root of the project.

# MAIN TASK
Your task is to generate the whole backend in the go programming language, to build this you have access to the database via mcp, you
will have to make the database structure, then run it on supabase, connect to it and start building the endpoints, after that you have
to connect those endpoint to the frontend, for all the endpoints you will have to write test and pass them, for the payments they 
should be idempotents, which means no matter how manu times a user clicks the button to pay, the charge should be applied only once,
for payments you have stripe for banking payments, for crypto payments you must accept bitcoin, ethereum and solana as crypto
payments, you also have to build all the catalog to render from the backend, save logo urls and send them to the frontend directly
from the backend, the backend must process everything and send to the frontend, the frontend must only renderize and do the less
possible ammounts of logic processing, that's why the backend is meant for.

# WHAT TO-DO
- Create database structure
- Create a whole backend, all endpoints for shopping, renderizing and payments
- Recommend platforms for crypto payments, because right now for banking payments we only have stripe
- Create cors and middleware, everything a backend must have
- Backend must run on 8000 port
- For all endpoints start with /api/v1. This is for having api versioning
- Document everything on ./docs/v1.0.0/v1.0.0.md
- Every request must be printed on terminal using log package from golang standard library
- All users must be or have the same permission for a normal online store, and there must be only one admin user who is gonna be the
one in charge on creating new merge and more

# WHAT NOT TO-DO
- Create or implemment large amounts of code without understanding the context of the application
- Execute weird sql queries without authorization
- Use extrange or not famous github golang libraries

# REFERENCES
You can have access to credential on .env.local file, for credential like postgres url of supabase, stripe credentials, etc.

# FEEDBACK
If you have any question feel free to ask via claude code terminal
