from flask import Flask, jsonify, request
import sqlite3
import uuid
import multiprocessing

app = Flask(__name__)

# Create an SQLite database
conn = sqlite3.connect('templates.db')
cursor = conn.cursor()

# Create the templates table
cursor.execute('''
    CREATE TABLE IF NOT EXISTS templates (
        template_id TEXT PRIMARY KEY,
        account_id TEXT NOT NULL,
        name TEXT,
        content TEXT
    )
''')
conn.commit()
conn.close()

# Helper function to execute SQL queries
def execute_query(query, data=None):
    conn = sqlite3.connect('templates.db')
    cursor = conn.cursor()
    if data:
        cursor.execute(query, data)
    else:
        cursor.execute(query)
    result = cursor.fetchall()
    conn.commit()
    conn.close()
    return result

# Function to start the Flask app on a specified port
def start_app(port):
    with app.app_context():
        app.run(debug=True, port=port, use_reloader=False)


# Create a new payment template
@app.route('/templates', methods=['POST'])
def create_template():
    data = request.get_json()
    account_id = data.get('account_id', '')
    name = data.get('name', '')
    content = data.get('content', '')

    if account_id and name and content:
        template_id = str(uuid.uuid4())
        # Insert the new template into the database with account_id
        execute_query("INSERT INTO templates (template_id, account_id, name, content) VALUES (?, ?, ?, ?)",
                      (template_id, account_id, name, content))
        return jsonify({'template_id': template_id, 'account_id': account_id, 'name': name}), 201
    else:
        return jsonify({'error': 'Account ID, name, and content are required'}), 400

# Retrieve all payment templates for a specific account_id
@app.route('/templates', methods=['GET'])
def get_templates_by_account():
    account_id = request.args.get('account_id', '')
    if account_id:
        templates = execute_query("SELECT template_id, name FROM templates WHERE account_id=?", (account_id,))
        template_list = [{'template_id': row[0], 'name': row[1]} for row in templates]
        return jsonify({'templates': template_list}), 200
    else:
        return jsonify({'error': 'Account ID is required'}), 400

# Retrieve a specific payment template by ID
@app.route('/templates/<string:template_id>', methods=['GET'])
def get_template(template_id):
    if template_id:
        template = execute_query("SELECT * FROM templates WHERE template_id=?", (template_id,))
        if template:
            return jsonify({'template_id': template[0][0], 'account_id': template[0][1], 'name': template[0][2], 'content': template[0][3]}), 200
    return jsonify({'error': 'Template not found'}), 404

# Update a payment template by ID
@app.route('/templates/<string:template_id>', methods=['PUT'])
def update_template(template_id):
    data = request.get_json()
    new_name = data.get('name')
    new_content = data.get('content')

    if template_id and (new_name or new_content):
        template = execute_query("SELECT * FROM templates WHERE template_id=?", (template_id,))
        if template:
            current_name = template[0][2]
            current_content = template[0][3]

            if new_name is None:
                new_name = current_name
            if new_content is None:
                new_content = current_content

            execute_query("UPDATE templates SET name=?, content=? WHERE template_id=?", (new_name, new_content, template_id))
            return jsonify({'message': f'Template with ID {template_id} updated successfully'}), 200
    return jsonify({'error': 'Template not found or no updates provided'}), 404

# Delete a payment template by ID
@app.route('/templates/<string:template_id>', methods=['DELETE'])
def delete_template(template_id):
    if template_id:
        template = execute_query("SELECT * FROM templates WHERE template_id=?", (template_id,))
        if template:
            execute_query("DELETE FROM templates WHERE template_id=?", (template_id,))
            return jsonify({'message': f'Template with ID {template_id} deleted successfully'}), 200
    return jsonify({'error': 'Template not found'}), 404

if __name__ == '__main__':
    # Number of replicas per service
    num_replicas = 3

    # Start each replica on a separate port
    for port in range(5005, 5005 + num_replicas):
        process = multiprocessing.Process(target=start_app, args=(port,))
        process.start()

    # Wait for processes to finish
    for process in multiprocessing.active_children():
        process.join()