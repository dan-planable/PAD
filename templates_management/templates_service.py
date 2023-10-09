
from flask import Flask, jsonify, request
import sqlite3
import uuid
import jwt
from functools import wraps

app = Flask(__name__)
app.config['SECRET_KEY'] = 'secret-key' 

# Create an SQLite database
conn = sqlite3.connect('templates.db')
cursor = conn.cursor()

# Create the templates table
cursor.execute('''
    CREATE TABLE IF NOT EXISTS templates (
        template_id TEXT PRIMARY KEY,
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

# Create a new payment template
@app.route('/templates', methods=['POST'])
def create_template():
    data = request.get_json()
    name = data.get('name', '')
    content = data.get('content', '')

    if name and content:
        template_id = str(uuid.uuid4())
        # Insert the new template into the database
        execute_query("INSERT INTO templates (template_id, name, content) VALUES (?, ?, ?)", (template_id, name, content))
        return jsonify({'template_id': template_id, 'name': name}), 201
    else:
        return jsonify({'error': 'Name and content are required'}), 400

# Retrieve all payment templates
@app.route('/templates', methods=['GET'])
def get_all_templates():
    templates = execute_query("SELECT template_id, name FROM templates")
    template_list = [{'template_id': row[0], 'name': row[1]} for row in templates]
    return jsonify({'templates': template_list}), 200

# Retrieve a specific payment template by ID
@app.route('/templates/<string:template_id>', methods=['GET'])
def get_template(template_id):
    if template_id:
        template = execute_query("SELECT * FROM templates WHERE template_id=?", (template_id,))
        if template:
            return jsonify({'template_id': template[0][0], 'name': template[0][1], 'content': template[0][2]}), 200
    return jsonify({'error': 'Template not found'}), 404\
        
# Update a payment template by ID
@app.route('/templates/<string:template_id>', methods=['PUT'])
def update_template(template_id):
    data = request.get_json()
    new_name = data.get('name')
    new_content = data.get('content')

    if template_id and (new_name or new_content):
        template = execute_query("SELECT * FROM templates WHERE template_id=?", (template_id,))
        if template:
            current_name = template[0][1]
            current_content = template[0][2]

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
    app.run(port= 5001,debug=True)