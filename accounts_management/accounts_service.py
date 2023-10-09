from flask import Flask, request, jsonify
import sqlite3
import uuid
import hashlib
import datetime

app = Flask(__name__)

# Create an SQLite database
conn = sqlite3.connect('accounts.db')
cursor = conn.cursor()

# Create the accounts table
cursor.execute('''
    CREATE TABLE IF NOT EXISTS accounts (
        account_id TEXT PRIMARY KEY,
        username TEXT NOT NULL UNIQUE,
        password_hash TEXT NOT NULL,
        balance REAL,
        transactions TEXT
    )
''')
conn.commit()
conn.close()

# Helper function to execute SQL queries
def execute_query(query, data=None):
    conn = sqlite3.connect('accounts.db')
    cursor = conn.cursor()
    if data:
        cursor.execute(query, data)
    else:
        cursor.execute(query)
    result = cursor.fetchall()
    conn.commit()
    conn.close()
    return result

# Create a new account
@app.route('/accounts', methods=['POST'])
def create_account():
    data = request.get_json()
    username = data.get('username', '')
    password = data.get('password', '')

    if username and password:
        password_hash = hashlib.sha256(password.encode()).hexdigest()

        # Check if the username already exists
        existing_account = execute_query("SELECT * FROM accounts WHERE username=?", (username,))
        if existing_account:
            return jsonify({'error': 'Username already exists'}), 400

        account_id = str(uuid.uuid4())  # Generate a unique account_id
        balance = 0.0
        transactions = []

        # Insert the new account into the database with account_id
        execute_query("INSERT INTO accounts (account_id, username, password_hash, balance, transactions) VALUES (?, ?, ?, ?, ?)",
                      (account_id, username, password_hash, balance, str(transactions)))

        return jsonify({'message': 'Account created successfully', 'account_id': account_id}), 201
    else:
        return jsonify({'error': 'Username and password are required'}), 400

# Get account balance
@app.route('/accounts/<string:account_id>/balance', methods=['GET'])
def get_balance(account_id):
    if account_id:
        account = execute_query("SELECT balance FROM accounts WHERE account_id=?", (account_id,))
        if account:
            return jsonify({'balance': account[0][0]}), 200
    return jsonify({'error': 'Account not found'}), 404

# Get account transactions
@app.route('/accounts/<string:account_id>/transactions', methods=['GET'])
def get_transactions(account_id):
    if account_id:
        account = execute_query("SELECT transactions FROM accounts WHERE account_id=?", (account_id,))
        if account:
            transactions = eval(account[0][0])
            return jsonify({'transactions': transactions}), 200
    return jsonify({'error': 'Account not found'}), 404

# Deposit funds into an account
@app.route('/accounts/<string:account_id>/deposit', methods=['POST'])
def deposit(account_id):
    data = request.get_json()
    amount = data.get('amount', 0)

    if amount > 0:
        account = execute_query("SELECT balance, transactions FROM accounts WHERE account_id=?", (account_id,))
        if account:
            balance = account[0][0]
            transactions = eval(account[0][1])
            balance += amount
            transactions.append(f"Deposited ${amount}")
            
            # Update the balance in the database
            execute_query("UPDATE accounts SET balance=?, transactions=? WHERE account_id=?", (balance, str(transactions), account_id))
            
            return jsonify({'message': f'Deposited ${amount} successfully'}), 201
    return jsonify({'error': 'Account not found or invalid amount'}), 404


# Withdraw funds from an account
@app.route('/accounts/<string:account_id>/withdraw', methods=['POST'])
def withdraw(account_id):
    data = request.get_json()
    amount = data.get('amount', 0)

    if amount > 0:
        account = execute_query("SELECT balance, transactions FROM accounts WHERE account_id=?", (account_id,))
        if account:
            balance = account[0][0]
            transactions = eval(account[0][1])
            if balance >= amount:
                balance -= amount
                transactions.append(f"Withdrew ${amount}")
                execute_query("UPDATE accounts SET balance=?, transactions=? WHERE account_id=?", (balance, str(transactions), account_id))
                return jsonify({'message': f'Withdrew ${amount} successfully'}), 201
            else:
                return jsonify({'error': 'Insufficient funds'}), 400
    return jsonify({'error': 'Account not found or invalid amount'}), 404

if __name__ == '__main__':
    app.run(debug=True)
