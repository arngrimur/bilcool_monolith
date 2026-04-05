bilcool_monolith is a booking application to creating bookings written in Golang. I need a U prompt for creating a UI.
a booking is a time slot in a calendar where a user can book a single vehicle
each boking also has a distance.
the UI shall be a web application.


- It shall use the defined endpoints in the API in the modules ../bookings, ../authentication, ../journal, and event_ledger. 
- The front-end shall use React as framework                                                                                                                                         
- Language shall be swedish or english and set by the user and be remembered by the UI
- The UI shall be responsive and mobile friendly
- It is possible to select light or dark mode and the choice shall be remembered by the UI                                                                                                                                                                                
- Users login using passkeys                                                                                                                                                                                                     
  - A user is created by an admin in before hand.                                                                                                                                                                                   
  - The first time a user logs in the user uses a OTP token, that can be requested and sent to the user e-mail.
  - The exist two types of users,
    - admin
    - user
  - The admin can create users and assign them roles.
  - Only a user defined by the admin can login and create, update or delete their bookings.

- A user can create a booking in 15 minutes intervalls. The start time must be after the end time.                                                                                                                               
- It is not possible to create a booking that ovelaps another booking.
- User can select a time slot in the calendar.
- A booking must be manually started and ended by the user when the booking starts and end.
- The user can cancel a booking by deleting the booking.
- The following views exist:
  - Calendar
    - month
    - week
    - day
  - Admin
    - list users
  - User profile
    - list user details of logged in user
  - Bookings, viewable by all users and admins
    - list of bookings for each user, start and end time and distance
    - list of bookings for each user in a specific month
    - as summary for each month of 
      - how many bookings were by the user
      - how many bookings were in total
      - how many km were booked