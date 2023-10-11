import unittest
import random
import string
from accounts_service import app

class TestAccountsService(unittest.TestCase):
    def setUp(self):
        self.client = app.test_client()

    def generate_random_username(self):
        return ''.join(random.choice(string.ascii_letters) for _ in range(10))

    def create_account(self, username):
        response = self.client.post('/accounts', json={'username': username, 'password': 'password'})
        created_account_data = response.get_json()
        self.assertEqual(response.status_code, 201)
        self.assertIn('account_id', created_account_data)
        return created_account_data['account_id']

    def assert_response_status(self, response, expected_status):
        self.assertEqual(response.status_code, expected_status)

    def test_create_account(self):
        username = self.generate_random_username()
        self.create_account(username)

    def test_get_balance(self):
        username = self.generate_random_username()
        account_id = self.create_account(username)
        response = self.client.get(f'/accounts/{account_id}/balance')
        self.assert_response_status(response, 200)

    def test_get_transactions(self):
        username = self.generate_random_username()
        account_id = self.create_account(username)
        response = self.client.get(f'/accounts/{account_id}/transactions')
        self.assert_response_status(response, 200)

    def test_withdraw(self):
        username = self.generate_random_username()
        account_id = self.create_account(username)
        response = self.client.post(f'/accounts/{account_id}/deposit', json={'amount': 100.0})
        self.assert_response_status(response, 201)
        response = self.client.post(f'/accounts/{account_id}/withdraw', json={'amount': 50.0})
        self.assert_response_status(response, 201)
        
    def test_insufficient_funds(self):
        username = self.generate_random_username()
        account_id = self.create_account(username)
        response = self.client.post(f'/accounts/{account_id}/withdraw', json={'amount': 50.0})
        self.assert_response_status(response, 400)

    def test_deposit(self):
        username = self.generate_random_username()
        account_id = self.create_account(username)
        response = self.client.post(f'/accounts/{account_id}/deposit', json={'amount': 100.0})
        self.assert_response_status(response, 201)

if __name__ == '__main__':
    unittest.main()
