# Improvements
- Improve comment/description format & parsing
- Used DB for previous results to remove data initialisation time

# Issues
- Failing http request causes fatal exit (TODO)
- Edge cases where a new item would not trigger notification (API issue)
    - Number of items for sales doesn't change because item is sold/added within check timeframe
- Can't check condition (API issue)
- Can't filter bad buyers (API issue)    
