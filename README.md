# PAD

**Idea:** A simple banking app which will consist of two microservices for User and Account Management.

### **Application Suitability:**

---

- Complex Functionality: A banking app typically consists of multiple independent modules, such as user authentication, account management, transaction processing, and reporting, making it well-suited for microservices.
- Scalability: Microservices allow each banking function to scale independently, ensuring efficient resource allocation.
- Fault Isolation: In financial applications, the importance of fault tolerance cannot be overstated. Microservices' fault isolation ensures that a failure in one part of the app doesn't compromise the entire system's integrity.
- Flexibility: Banking apps may require diverse technologies, such as security frameworks, payment gateways, and reporting tools. Microservices allow you to select the most suitable technologies for each service.
- Real-world Examples: Successful companies like PayPal employ microservices to handle the complexity and scalability demands of financial transactions.

### Why choose a Microservices approach over a Monolithic for a Banking App:

---

- Complexity:

Monolithic: In a monolithic architecture, the entire banking application is contained within a single codebase and can become complex to manage as the app grows in features and functionalities.

Microservices: Microservices break down the app into smaller, more manageable units, reducing the complexity of each individual service.

- Scalability:

Monolithic: Scaling a monolithic banking app often involves replicating the entire application, which can lead to inefficient resource usage.

Microservices: Microservices enable you to scale specific components, such as transaction processing or user authentication, independently based on their resource needs, leading to efficient resource allocation.

- Technology Stack:

Monolithic: In a monolithic architecture, you are limited to using a single technology stack for the entire banking app, which may not be the best choice for every component.

Microservices: Microservices allow you to select the most suitable technology stack for each service, optimizing performance and development speed. For example, you can use Python/Flask for one service and Golang for another.

- Fault Tolerance:

Monolithic: A failure in one part of a monolithic banking app can potentially bring down the entire application.

Microservices: Microservices are designed for fault isolation, meaning that a failure in one service does not necessarily impact the operation of other services, improving overall system resilience.

### Architecture:

---

![The Architecture Diagram](https://github.com/dan-planable/PAD/blob/main/Assets/Architecture.drawio.png)

### Service Boundaries:

---

- User Service - Responsible for user registration, login, and authentication.

- Account Management Service - Handles user account creation, balance inquiries, transactions, and related operations.

- API Gateway - Has the role on an entry point for requests from the client-side. Each request being routed to the respective service.

- Service Discovery - Keeps a registry of existing service addresses and helps knowing where each instance is located.

- Load Balancer - Responsible for distributing incoming requests among multiple instances of the same service and makes sure of an even distribution of requests. Works on the basis of a Round Robin.

- Database - Serves as a Data Storage and is responsible for storing application data.

### Technology Stack and Communication Patterns:

---

- API Gateway, Service Discovery and Load Balancer:

  **Technology stack** - Go.

  **Communication Pattern** - REST API for Gateway.

- User and Account Management Services :

  **Technology stack** - Python with Flask.

  **Communication Pattern** - REST API.

### Data Management:

---

**User Service Endpoints:**
Responsible for user registration, login, and authentication.
Exposes endpoints such as **/register**, **/login**, and **/profile/{user_id}**.

**POST /register**

```json
{
  "username": "dan",
  "email": "dan@gmail.com",
  "password": "myPassword"
}
```

```json
{
  "userId": "1",
  "username": "dan",
  "email": "dan@gmail.com"
}
```

**POST /login**

```json
{
  "username": "dan",
  "password": "myPassword"
}
```

```json
{
  "userId": "1",
  "acccessToken": "123..."
}
```

**GET /profile/{user_id}**

```json
{
  "userId": "1",
  "username": "dan",
  "email": "dan@gmail.com"
}
```

Account Management Endpoints: Handles user account creation, balance inquiries, transactions, and related operations. Provides endpoints like **/accounts**, **/accounts/{account_id}**, **/accounts/{account_id}/balance**, **/accounts/{account_id}/transactions**, **/accounts/{account_id}/deposit**

**POST /accounts**

```json
{
  "user_id": "1",
  "account_type": "savings",
  "initial_balance": 1000.0
}
```

```json
{
  "account_id": "1",
  "user_id": "1",
  "account_type": "savings",
  "balance": 1000.0,
  "created_at": "2023-09-30T12:00:00Z"
}
```

**GET /accounts/{account_id}/balance**

```json
{
  "account_id": "1",
  "user_id": "1",
  "account_type": "savings",
  "balance": 1000.0
}
```

**GET /accounts/{account_id}/transactions**

```json
{
  "account_id": "1",
  "user_id": "1",
  "transactions": [
    {
      "transaction_id": "1",
      "type": "deposit",
      "amount": 500.0,
      "timestamp": "2023-09-30T14:30:00Z"
    },
    {
      "transaction_id": "2",
      "type": "withdrawal",
      "amount": 200.0,
      "timestamp": "2023-09-30T15:15:00Z"
    },
    {
      "transaction_id": "3",
      "type": "deposit",
      "amount": 100.0,
      "timestamp": "2023-09-30T14:40:00Z"
    }
  ]
}
```

**POST /accounts/{account_id}/deposit**

```json
{
  "amount": 100.0
}
```

**POST /accounts/{account_id}/withdraw**

```json
{
  "amount": 100.0
}
```

### Deployment and Scaling:

- For deployment I'll use Docker as I already had experience with it in previous courses.
