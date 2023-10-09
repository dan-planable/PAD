from flask import Flask, request, jsonify
import sqlite3
import uuid
import hashlib
import jwt
import datetime
from functools import wraps

app = Flask(__name__)
app.config['SECRET_KEY'] = 'secret-key' 

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

def token_required(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        token = request.headers.get('Authorization')

        if not token:
            return jsonify({'error': 'Token is missing'}), 401

        try:
            data = jwt.decode(token, app.config['SECRET_KEY'], algorithms=['HS256'])
            current_user = data['username']
        except jwt.ExpiredSignatureError:
            return jsonify({'error': 'Token has expired'}), 401
        except jwt.InvalidTokenError:
            return jsonify({'error': 'Invalid token'}), 401

        return f(current_user, *args, **kwargs)

    return decorated

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

        account_id = str(uuid.uuid4())
        balance = 0.0
        transactions = []

        # Insert the new account into the database
        execute_query("INSERT INTO accounts (account_id, username, password_hash, balance, transactions) VALUES (?, ?, ?, ?, ?)",
                      (account_id, username, password_hash, balance, str(transactions)))

        token = jwt.encode({'username': username, 'exp': datetime.datetime.utcnow() + datetime.timedelta(hours=1)},
                           app.config['SECRET_KEY'], algorithm='HS256')

        return jsonify({'message': 'Account created successfully', 'token': token}), 201
    else:
        return jsonify({'error': 'Username and password are required'}), 400

# Authentication endpoint to get a token
@app.route('/login', methods=['POST'])
def login():
    data = request.get_json()
    username = data.get('username', '')
    password = data.get('password', '')

    if username and password:
        # Retrieve the user from the database
        user = execute_query("SELECT * FROM accounts WHERE username=?", (username,))
        if user:
            user = user[0]
            password_hash = hashlib.sha256(password.encode()).hexdigest()
            if user[2] == password_hash:
                token = jwt.encode({'username': username, 'exp': datetime.datetime.utcnow() + datetime.timedelta(hours=1)},
                                   app.config['SECRET_KEY'], algorithm='HS256')
                return jsonify({'message': 'Login successful', 'token': token}), 200

    return jsonify({'error': 'Invalid username or password'}), 401

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
