# Simulated mailbox presentation

A responsive, dependency-free mailbox UI for demonstrating the employee side of the security-training flow.

## Run locally

From the repository root:

```sh
python3 -m http.server 4173 --directory presentation
```

Then open <http://localhost:4173>.

## Demo flow

1. Browse or search the synthetic inbox.
2. Open **“We've scheduled your Google data download”**.
3. Select **“Review activity”** to reveal the simulated-phishing training feedback.
4. Use the folder, unread, label, star, compose, and responsive navigation interactions during the presentation.

## Privacy and safety

- Subjects mirror a small set of locally exported emails to make the inbox categories recognizable.
- Senders' addresses use the reserved `.example.test` domain.
- Previews, message bodies, timestamps, reservation references, and account details are synthetic.
- No original message body, recipient address, bank/card/account detail, booking identifier, authentication link, or tracking URL is included.
- The mailbox has no network dependency and does not collect or transmit form values or interaction events.
