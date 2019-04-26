## Categories

- membership: a fixed rate is charged to the users for a fix interval, e.g. $7/month membership, or $40/annual.
- upsell: grouped sets of features are available as options and are charged differently
- limited supply: similar to the upsell, but with limited stocks for each plans
- build your own plan: assigns a price for each feature, they are charged independently and the users may choose the features they only want to purchase
- addons - user can choose to upgrade the plans with a better plans at an additional charge
- pay per use - users pay for what they use only. usually this plans will have a list of services that are available (think of AWS)
- graduated pay per user. similar to the pay-per-use, but discounted for heavy users. 
- utilization - commonly seen in datacenters, the utilization of the service is tracked and the users are charged accordingly by 95th percentile or average usage.


## Thoughts

- does the start/end date matters? If I subscribed in the middle of the month, will I get charged the full amount or partial (prorated?)
- if I cancel the subscription, do I get the refund for the remaining month?
- does the subscription have a cooldown period (1 month means the plan will end at the month, regardless of when it is cancelled), or is it immediate (terminate now, and it will take effect immediately)
- if the plan is upgraded, will the previous term (valid from to valid till) still be valid, or will they be extended?
- plans can be extended, upgraded, terminated, term started, term ended (expired), transferred etc
- who can purchase subscription? preferable a party, since a party can be either an organization or person, and the subscription can be for a person or organization. If the subscription is applied to an organization, then the employees belonging to the organzation will all benefit from it

## Subscription schema

- https://www.nathanhammond.com/the-subscription-library-schema-to-rule-them-all
- https://www.nathanhammond.com/patterns-for-subscription-based-billing

## Subscription plans naming

https://www.paidmembershipspro.com/how-to-name-your-membership-levels-or-subscription-options/
