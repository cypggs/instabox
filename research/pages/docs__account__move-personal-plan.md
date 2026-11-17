> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Move a personal plan into a workspace

> Move your personal sandboxes, credits and plan into a workspace you own, without stopping anything.

If you paid for a plan on your personal account and now work in a workspace, you can move everything
into the workspace in one step. You don't need to cancel the personal plan first. Cancelling first would
pause your sandboxes when the plan ends.

```bash theme={null}
instabox workspace migrate acme
```

You must own the workspace. What happens to your personal plan depends on the workspace:

* **The workspace has a paid plan of its own.** Your personal plan ends, as described below.
* **The workspace has no plan, or the move makes a new one.** Your plan moves with you. See
  [Your plan moves with you](#your-plan-moves-with-you).

The move is available to admins first, while it rolls out.

## What moves

* **Sandboxes.** A running sandbox is paused, moved and resumed. It keeps its memory and processes,
  and stops responding for a few seconds. From then on, the workspace pays for it, including a
  sandbox that ran free on your personal Free plan.
* **Snapshots, Secrets and API keys.** Secrets are encrypted again for the workspace. The workspace's
  owners and admins can see and use everything that moves.
* **Credits.** Every credit grant that has not expired and has money left moves, startup credits
  included. Each grant keeps its balance and its expiry date. Spent and expired grants stay on your
  personal account.
* **The plan, into a workspace with its own plan.** Your personal plan ends today, with no card
  refund. The unused part of this month's fee becomes credit in the workspace:
  * If you paid the plan with credits, the money goes back into the grants it came from, with their
    expiry dates. A grant that has expired since then gets nothing back.
  * If you paid the plan by card, it becomes a new credit grant that expires one year after the move.

Your personal account stays, on the Free plan.

## Your plan moves with you

When the workspace has no plan of its own (a new workspace, or one whose plan ended), your active
personal plan moves into it as it is. Nothing is canceled, refunded or charged for the move:

* The workspace is now on your plan's tier, paid the same way: by the same card and Stripe customer,
  or from the credits that move with you.
* The renewal date, the invoice history and any change you scheduled (a lower plan or a cancellation
  at the end of the month) stay as they were. The next renewal bills the workspace.
* This month's allowance stays one allowance. What your personal account used so far counts against
  it, and what is left moves to the workspace's month. If your personal account already used more than
  the allowance, that part is billed at the move, credits first and then the card, as it would be at
  the end of the month.
* A plan paid with credits renews on the 1st from the workspace's credits, which now include yours.
* The workspace uses your card and your Stripe billing account, so only you can open its billing
  portal. The workspace's other admins can't.
* Your old invoices stay with the workspace's billing account. If you buy a personal plan later, it
  gets a new billing account of its own.

While the move runs (a minute or two), changing either plan or buying one for the workspace
waits: retry in a minute. Your plan doesn't move while it owes something, such as an unpaid usage invoice: pay it first. If
the card declines the usage invoiced at the move, nothing moves. If the move fails after that
usage was billed, it stays billed (you had already used it), your plan stays on your personal
account, and the rest of the month is billed at its renewal as usual. A workspace that still owes money
for an earlier plan, or that an admin put on a plan without a subscription, can't take your plan.
Pay what it owes, or ask support.

## See what would move first

To see what would move and how much credit carries over, without changing anything, run:

```bash theme={null}
instabox workspace migrate acme --dry-run
```

The output looks like this:

```
Would move into acme: 2 sandboxes (1 running: paused for a few seconds, then resumed),
1 credit grant with their expiry dates. Your personal pro plan ends today; its unused $12.50
this month becomes workspace credit.
```

Without `--dry-run`, the command shows the same summary and asks you to confirm. To skip the question,
add `--yes`.

In the dashboard, open the workspace's **Billing** page and select **Move Personal Plan Here**.

## Move into a new workspace

If you don't own a workspace yet, the move can make one for you:

```bash theme={null}
instabox workspace migrate --new acme --dry-run
instabox workspace migrate --new acme
```

If you have an active personal plan, it moves into the new workspace as described in
[Your plan moves with you](#your-plan-moves-with-you): the same tier, card and renewal date, and nothing
is charged. The dry run says so:

```
Would move into a new workspace acme: 1 sandbox. Your pro plan moves with you, with its card and
renewal date, and the $17.00 left of this month's allowance. Nothing is canceled or refunded, and
nothing is charged.
```

Without a personal plan, the new workspace starts on the plan you pick with `--plan`. Its fee for the
rest of this month comes out of the credits that move with you, so you don't need a card. From then
on it renews from its credits each month, like any workspace paid with credits; add a card before
they run out.

If your credits don't cover that first fee, nothing moves. The command prints a Checkout link to buy
the workspace instead. Once it's paid, run `instabox workspace migrate acme` to move into it.

In the dashboard, open your personal **Billing** page and select **Move to a New Workspace**.

> [accordion] When the move is refused
>
> The move does not start, and nothing changes, when:
>
> * your personal plan has an unpaid invoice, or its first payment is not complete
> * a checkout of your personal account is still open
> * a sandbox is starting, stopping, or has another operation in progress
> * a sandbox does not pause within 2 minutes
>
> Wait a moment, then run the command again. If a step fails after your sandboxes and credits
> moved, the command says so. Run the same command again to finish, in the same workspace: nothing
> is carried over twice, and no second workspace is made.

> [accordion] Over the API
>
> `POST /v1/orgs/{id}/billing/move-personal` with `{"dryRun": true}` reports what would move.
> `POST /v1/orgs/move-personal` with `{"name": "acme"}` moves into a new workspace. Both need a
> signed-in session: API keys cannot call them. When the plan moves, the reply has `movedTier`.
> Admins can move a user's personal account with
> `POST /v1/admin/users/{id}/move-personal` (`{"into": "org_..."}` or `{"newWorkspace": "acme"}`), or
> `instabox admin users move-personal`. See the [API reference](https://instabox.sh/docs/api-reference/introduction.md).

> [card] [Billing and usage](https://instabox.sh/docs/account/billing.md)
>
> This month's cost and your credit balance.

> [card] [Plans and pricing](https://instabox.sh/docs/account/pricing.md)
>
> What each plan includes.
