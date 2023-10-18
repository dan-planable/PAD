# PAD

## How to test application

I don't exactly know what should be done after downloading images from Docker Hub but if you start them one by one then this should be the order:

1.  Accounts Management
2.  Templates Management
3.  Service Directory
4.  Gateway

(not sure that redis image was really required to upload on Hub so everything should run without it)

I also uploaded to GitHub the Postman Collection with all requests for the app, it can be downloaded and used. Otherwise, below are all the endpoints with request bodies. (no auth or special behaviour)

**Idea:** A simple banking app which will consist of two microservices for Templates and Account Management.

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

- Templates Service - Responsible for managing all payment templates, creating, retrieving, updating and deleting them.

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

**Templates Service Endpoints:**
Responsible for templates management, contains endpoints such as **/templates**, **/templates/template_id**.

**POST /templates**

```json
{
  "name": "Water",
  "content": "Payment for electricity",
  "account_id": "e70ae4cb-3dd1-4f8d-9a14-3ce8e2b61917"
}
```

```json
{
  "account_id": "e70ae4cb-3dd1-4f8d-9a14-3ce8e2b61917",
  "name": "Water",
  "template_id": "2d5ac173-ef43-4d2f-a52a-edc79c26e698"
}
```

**GET /templates?account_id**

```json
{
  "templates": [
    {
      "name": "Water",
      "template_id": "0bb57d9f-b284-4878-bfda-cafaf8e5c3dc"
    },
    {
      "name": "Electricity",
      "template_id": "42a9b9cf-25dd-4db0-a71a-ff26b7c625dd"
    }
  ]
}
```

**GET /templates/{template_id}**

```json
{
  "account_id": "e70ae4cb-3dd1-4f8d-9a14-3ce8e2b61917",
  "content": "Payment for electricity",
  "name": "Water",
  "template_id": "0bb57d9f-b284-4878-bfda-cafaf8e5c3dc"
}
```

**PUT /templates/{template_id}**

```json
{
  "name": "New Electricity Bill",
  "content": "New Payment for electricity"
}
```

```json
{
  "message": "Template with ID 2d5ac173-ef43-4d2f-a52a-edc79c26e698 updated successfully"
}
```

**DELETE /templates/{template_id}**

```json
{
  "message": "Template with ID 2d5ac173-ef43-4d2f-a52a-edc79c26e698 deleted successfully"
}
```

Account Management Endpoints: Handles user account creation, balance inquiries, transactions, and related operations. Provides endpoints like **/accounts**, **/accounts/{account_id}/balance**, **/accounts/{account_id}/deposit**, **/accounts/{account_id}/withdraw**, **/accounts/{account_id}/transactions**,

**POST /accounts**

```json
{
  "username": "dan11nnn11nnss",
  "password": "pa1ssword"
}
```

```json
{
  "account_id": "9481da99-dd0f-43ff-bc89-9d352a76aa08",
  "message": "Account created successfully"
}
```

**GET /accounts/{account_id}/balance**

```json
{
  "balance": 100.0
}
```

**GET /accounts/{account_id}/transactions**

```json
{
  "transactions": [
    "Deposited $69",
    "Deposited $69",
    "Withdrew $10",
    "Withdrew $10"
  ]
}
```

**POST /accounts/{account_id}/deposit**

```json
{
  "amount": 69
}
```

```json
{
  "message": "Deposited $69 successfully"
}
```

**POST /accounts/{account_id}/withdraw**

```json
{
  "amount": 10.0
}
```

```json
{
  "message": "Withdrew $10 successfully"
}
```

### Deployment and Scaling:

- For deployment I'll use Docker as I already had experience with it in previous courses.
