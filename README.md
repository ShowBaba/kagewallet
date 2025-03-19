# KageWallet Telegram Bot

KageWallet is a robust Telegram bot designed for seamless cryptocurrency trading. It offers a user-friendly interface, secure transactions, and essential features for buying, selling, and managing crypto assets directly within Telegram.

---

## Features

### 📌 Core Features
- **Sell Cryptocurrency**: Convert cryptocurrencies like USDT, TRON, and SOL into Naira and withdraw to local bank accounts.
- **Live Exchange Rates**: Access real-time exchange rates for informed trading decisions.
- **Wallet Management**: Check balances and transaction history with simple commands.
- **Withdraw Funds**: Seamlessly transfer funds from the wallet to a bank account.
- **Security**: Multi-level authentication with password protection for sensitive actions.

### 🛠 Additional Features
- **Command-Based Navigation**: Use intuitive commands for all bot operations.
- **Session Management**: Refresh sessions and update data instantly.
- **Password Reset**: Reset forgotten passwords with ease.
- **Transaction History**: Track all deposits, trades, and withdrawals.

---

---

## Technology Stack

- **Golang** – Backend logic and Telegram bot implementation.
- **[Blockradar.co](https://www.blockradar.co/)** – Handles cryptocurrency transactions and exchange rates.
- **[Monnify.com](https://www.monnify.com/)** – Processes fiat withdrawals and bank payouts.
- **PostgreSQL** – Stores user balances, transactions, and authentication data.
- **Redis** – Caches user sessions for improved performance.

---

## Commands

| Command              | Description                                           |
|----------------------|-------------------------------------------------------|
| `/start`             | Start the bot and see the list of commands.           |
| `/sell`              | Sell cryptocurrency and receive payment in fiat.      |
| `/help`              | See a list of available commands and their descriptions. |
| `/set_password`      | Set a new password for your account.                  |
| `/reset_password`    | Reset your password if forgotten.                     |
| `/refresh`           | Refresh your session and update data.                |
| `/rate`              | Get the current exchange rate.                        |
| `/balance`           | Check your balance for a specific asset.             |
| `/transactions`      | View your complete transaction history.               |
| `/withdraw`          | Withdraw funds to your bank account.                 |

---

## How It Works

1. **Start the Bot**  
   Type `/start` to activate the bot and explore its features.

2. **Set Up Password**  
   Secure your account by setting a password using `/set_password`. This password is required for sensitive actions.

3. **Deposit Crypto**  
   Use the generated wallet addresses to deposit supported cryptocurrencies.

4. **Trade or Withdraw**
    - Use `/sell` to trade crypto for Naira.
    - Use `/withdraw` to transfer funds to your bank account.

5. **Check Rates**  
   Stay updated with real-time exchange rates using `/rate`.

6. **Track Activity**  
   View balances, transaction history, and more with intuitive commands like `/balance` and `/transactions`.

---

## Installation

### Prerequisites
- **Golang**: Ensure you have Go installed on your system.
- **Telegram Bot Token**: Obtain your bot token from [BotFather](https://core.telegram.org/bots#botfather).

### Steps
1. Clone the repository:
   ```bash
   git clone https://github.com/ShowBaba/kagewallet.git
   cd kagewallet
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Run the bot:
   ```bash
   // dev
   ENV=dev go run main.go
   
   // live
   go run main.go
   ```

---

## Configuration

Ensure all required environment variables are properly set as specified in `.env.example`:

---

## Security

- **Password Protection**: All sensitive operations require user authentication.
- **Secure Data Storage**: User data is encrypted and stored securely.
- **Real-Time Monitoring**: Sessions and activities are tracked to prevent unauthorized access.

---

## Contribution

1. Fork the repository.
2. Create a feature branch:
   ```bash
   git checkout -b feature-name
   ```
3. Commit changes:
   ```bash
   git commit -m "Description of changes"
   ```
4. Push to the branch:
   ```bash
   git push origin feature-name
   ```
5. Open a pull request.

---

