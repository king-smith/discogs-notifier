# Discogs Notifier
Discogs notifier is an application used to email the user when new listings have been added to discogs. Discog's inbuilt notification system is a daily email limited to 100 items so you may miss new items due to infrequency and/or lack of brevity.

## Usage

### Setup
Install dependencies

`go dep`

Create `.env` file

`cat .env.example > .env`

Fill in `.env`
- `CURRENCY`: Currency definition (Options found [here](https://www.discogs.com/developers#page:marketplace,header:marketplace-release-statistics))
- `DISCOGS_USERNAME`: Username of discogs account
- `DISCOGS_TOKEN`: User token of discogs account (Create [here](https://www.discogs.com/settings/developers)
- `SMTP_USERNAME`: Username of smtp client account
- `SMTP_PASSWORD`: Password of smtp client account
- `SMTP_ADDRESS`: Address of SMTP client
- `USER_EMAIL`: Email of notification recipient

Create a user list in discogs with the tag `notify_me` in the description

Add minimum prices as an item comments if desired (no comment == no minimum price). e.g.

`30.50`

### Run
`go run main/main.go`

## Improvements
- Improve comment/description format & parsing
- Used DB for previous results to remove data initialisation time

## Issues
- Edge cases where a new item would not trigger notification (API issue)
    - Number of items for sales doesn't change because item is sold/added within check timeframe
- Can't check condition (API issue)
- Can't filter bad buyers (API issue)    
