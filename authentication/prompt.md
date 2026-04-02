Build a microservice backend, for creating and managing user accounts, that handles authentication, i.e. login using passkeys.

# User creation
- A user can be created by by an admin. input is a username and an e-mail.  
- When a user is created a Sns notice is sent on the topic "users" with type "created".
- When a user is deleted a Sns notice is sent on the topic "users" with type "deleted".

There exist helper functions for creating Sns notices in module message_broker in the package sns. 
See one directory up. Also look at bookings directory for an example. 

# User login
- A user can login using a passkey on any device. 
- If a passkey does not exist
  -  The user enters his e-mail 
    - if the e-mail is the same as the registerd e-mail 
      - a 6 digits security token is sent to the mail address of the user.
      - The user enters the security token.
      - the passkey is created and the user is registered.
- A security token is valid for 10 minutes.

# REST services
The backend should provide the following REST services:
- POST /users
- POST /users/login

# Technoloys
- Passkeys
- Golang 1.26.0
- Postgres 18.0

# Teststrategie
- All code shall be unit tested.
- There shall integration tests towards the database.
- There shall be system test that checks the login functionality.

There exist a testing module in the package testing for setting up the database and localstack.
See the  booking directory for an example.