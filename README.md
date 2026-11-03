# Cells

**A personal task manager with email and Discord reminders.**

Cells helps you organize your work, identify priorities, and track completed tasks. Tasks are grouped by priority so you can quickly find the work that requires your attention.

Use Cells to manage daily work, coursework, or personal plans. Host it on your own computer or server and access it through a browser.

## Features

### Organize daily tasks

Create tasks with a title, notes, category, priority, and due date. Update the details as your plans change, mark tasks as completed, or delete tasks you no longer need.

Below the due date, choose **Repeat**: **5 Days** (Monday–Friday), **7 Days** (every day), or **Custom days**. Presets lock their selected days. Completed tasks reopen on the next selected day in Asia/Jakarta, once their due date has arrived, and receive that day's due date. Incomplete tasks stay open; missed occurrences do not create duplicates. Refresh the list to see tasks reopened by the server.

### Focus on priorities

Tasks are grouped into high, medium, and low priorities. Search and category filters help you locate specific items, while overdue indicators and completion progress show what remains to be done.

### Receive reminders

Send summaries of active tasks by email or to a Discord channel through a webhook. Send messages immediately or schedule daily delivery at a selected time in the Asia/Jakarta time zone (WIB, UTC+7).

Enable the **Notify** switch on each task to include it in notification templates and totals. It defaults to off, including for existing tasks. Only incomplete tasks with Notify enabled are included; plain messages and test messages do not use task data.

Email supports multiple recipients, plain text, and HTML. Discord messages support text and Markdown. Each channel has its own message and schedule settings.

### Customize messages

Use plain messages or templates that include task information, such as your name, the number of active tasks, and task lists grouped by priority. Preview messages and send tests before enabling scheduled delivery. A default HTML table template is available for email.

### Keep accounts separate

Each account has its own tasks, categories, and notification settings. Multiple accounts can use the same installation with separate task lists.

Sign-in sessions expire after six hours. Users must then sign in again; activity and page refreshes do not extend the session.

## Example workflow

Start your day by adding tasks and assigning priorities. Organize them into categories such as **Study**, **Work**, or **Personal**, and set due dates. Mark tasks as completed to update your progress.

For regular reminders, open **Settings → Notifications**, choose email or Discord, and save your message and delivery schedule. For example, send a summary of active tasks every day at **08:00 WIB**. The server must remain running for scheduled delivery.

## Run your own instance

Cells uses React, Go, and PostgreSQL. Docker Compose is the supported deployment method.

1. Install Docker and Docker Compose. On Windows, start Docker Desktop.
2. Copy `.env.example` to `.env` and set `DB_PASSWORD`.
3. Run the following command from the project directory:

```bash
docker compose up -d --build
```

Open [http://localhost:8080](http://localhost:8080). The initial account uses **admin** as both the username and password. Replace the default password with your own password containing at least eight characters:

```bash
docker compose exec backend /app/server passwd admin "yourNewPassword"
```

For email, enter your SMTP configuration in `.env`, then run `docker compose up -d backend`. In Notifications, separate recipient addresses with semicolons, for example `first@example.com; second@example.com`. For Discord, save your webhook URL in the Discord notifications panel.

Application data is stored in the PostgreSQL volume. The `.env` file contains private configuration and is excluded from Git.
