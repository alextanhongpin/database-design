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



Date scenario:
- user is charged only at the end of the month (charging at the start of the month means more refunds logic, since they need to be refunded for what they didn't pay for)
- assume user select monthly subscription
- user subscribe at a date other than start of the month
	- option 1: charge user only for the rest of the month
	- option 2: if the days are less than 5 (?), give them for free
	- option 3: user cancel immediately (?) if day == 0, exclude charges
- user cancel subscription
	- option 1: charge user only for the period they used from the start of the month until the date they cancel the subscription
	- option 2: no refund
- user upgrade subscription
	- charge the difference at the end of the month
- user downgrade subscription
	- charge the difference at the end of the month

Role Scenario:
- organization vs user purchase subscription
	- different plans for user and organization, different costs and different rules

Subscription Plan Scenario:
- introducing new plans
	- don't update the existing data in db, create new plan so that old users will maintain their subscription
	- automatically upgrade user's plan?
- introducing new feature?

Rest:
- does the start/end date matters? If I subscribed in the middle of the month, will I get charged the full amount or partial (prorated?)
- if I cancel the subscription, do I get the refund for the remaining month?
- does the subscription have a cooldown period (1 month means the plan will end at the month, regardless of when it is cancelled), or is it immediate (terminate now, and it will take effect immediately)
- if the plan is upgraded, will the previous term (valid from to valid till) still be valid, or will they be extended?
- plans can be extended, upgraded, terminated, term started, term ended (expired), transferred etc
- who can purchase subscription? preferable a party, since a party can be either an organization or person, and the subscription can be for a person or organization. If the subscription is applied to an organization, then the employees belonging to the organzation will all benefit from it. TL;DR there can be both individual and organization plan.
- subscription plans can be upgraded/downgraded/cancelled anytime (unless there is a minimum period). There may be additional costs (prorated) when upgrading in the middle of the month, or refund when downgrading/cancelling it. The features will also change (how do we detect the changes? when the user login and obtain a jwt token? but how to handle changes in that?)
- is there a mandatory minimum for the plans?
- will user receive freemium? we can create a default plan that is freemium (valid say for 1 month), which cannot be renewed.
- statuses: subscriptions can be renewed, cancelled, upgraded, downgraded etc
- plans are valid for the period (month, year, etc)
- plans have a start and end date
- subscriptions are charged at the end of the month
- if the users did not paid for the month, next month account is temporarily disabled
- if the users downgrade the plan, the features will be missing (disabled), when they enable it back then the features will be added back. additional costs are refunded (?) unless stated not in the agreement
- how to renew the existing subscription? ( without plan changes)
- Do we need to create the basic plan? It will only take up rows in the db.
- How to pause subscription?
- How to generate invoice for subscription?
- What if the plans changed? Add new plans, expire the old one through valid from date, but don’t delete it. The old data may still reference the old plan, but the new ones will have the new plans. What if we want to force upgrade the old plans (deprecation), we can automatically extend them.
- the period (weekly, monthly, yearly) matters if we are going to do deduction/refund when the user changes the subscription plan
- the cost per day, month, year also needs to be defined
- is the cost affected by other rules? such as country, location, roles.
- if the user upgrades his subscription, then he has to pay more for the current elapsed difference in the duration left for the current subscription. What if the user downgrades his subscription? Do we need to refund the subscription?
- If the plans is for individual vs organization, the features could have different business rules, e.g. individual can create 5 items. But if we have organization account, we can probably have a rule that only 5 users can be added, and each of them can only create 1 item.

## Subscription schema

- https://www.nathanhammond.com/the-subscription-library-schema-to-rule-them-all
- https://www.nathanhammond.com/patterns-for-subscription-based-billing
- https://softwareengineering.stackexchange.com/questions/196524/handling-subscriptions-balances-and-pricing-plan-changes

## Subscription plans naming

https://www.paidmembershipspro.com/how-to-name-your-membership-levels-or-subscription-options/


## Sample code in Golang

```go
package main

import (
	"fmt"
	"math"
	"time"
)

type Feature struct {
	ClientCount int
	// The cost of the feature.
	// NOTE: What is the currency of the cost?
	Cost float64
}

type Plan struct {
	ID   string
	Type string

	// The date the plan is introduced.
	ValidFrom time.Time

	// The validity of the plan. If we want to terminate the plan, just set the valid till date.
	ValidTill time.Time

	// The Plan that the current plan superseded. This may happen when we deprecate the old plans and introduce a new one.
	ParentID string

	// Cost Overwrite - either take the cost of the plans, or indicate the overwritten cost here in the Plan.
	// NOTE: For fremium, we probably also need to generate an invoice for the user. That way, we can probably keep track of the costing model for freemium. (or not).
	CostPerDay   float64
	CostPerMonth float64
	CostPerYear  float64
	// The country the plan is available in.
	Country string
	// The currency used for the plan pricing. For conversion, we can create a pricing table for the costs.
	Currency string
}

// The plan features keep track of the features for each plans. Different plans that are in different countries, region, tier may have different plan features available.
// Whenever a new plan is created/deprecated/deleted/updated, the plan features needs to be modified as well.
type PlanFeature struct {
	PlanID   string
	Features []Feature
}

type SubscriptionPlan struct {
	ID             string
	SubscriptionID string
	PlanID         string
	ValidFrom      time.Time
	ValidTill      time.Time
	PeriodType     string // weekly, monthly, annually.
	// The previous subscription that is renewed. Note that the fremium model cannot be renewed.
	ParentID string
	// A boolean to keep track of the paid status. If the previous subscription is not paid, do not extend.
	// Once the user made the payment, update the status.
	IsPaid      bool
	IsRenewable bool

	// NOTE: We might need the following to compute the final cost of the subscription,
	// since different country might have different pricing tables.
	// On second thoughts, probably create the different plans in different countries. The plan tables won't be that much anyway.
	// Country string
	// Currency string
}

// Prorated cost of the subscription.
func (s *SubscriptionPlan) CalculateCost(plans map[string]Plan) float64 {
	days := math.Ceil(s.ValidTill.Sub(s.ValidFrom).Second() / (24 * time.Hour))
	// TODO: Handle if plans does not exist.
	currentPlan := plans[s.PlanID]
	return days * currentPlan.CostPerDay
}

type Invoice struct {
	SubscriptionPlanID string
	Amount             float64
	// The date the invoice is sent.
	SentAt time.Time
	// The date the invoice id paid.
	PaidAt time.Time
	UserID string
}

type User struct {
	ID string
}

type Subscription struct {
	ID     string
	UserID string
	// The date the subscription is made.
	ValidFrom time.Time
	// The date the subscription is terminated.
	ValidTill time.Time
	// A boolean to indicate if the subscription is still active.
	Active bool
	// If true, then the subscription will be renewed automatically.
	AutoRenewed bool
}

func main() {
	// Assuming we already have a user.
	u := User{"1"}
	// And a bunch of plans.
	p0 := Plan{
		ID:        "0",
		PlanType:  "freemium",
		ValidFrom: time.Now(),
		ValidTill: time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
	}
	p1 := Plan{
		ID:        "1",
		PlanType:  "basic",
		ValidFrom: time.Now(),
		ValidTill: time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
	}
	p2 := Plan{
		ID:        "2",
		PlanType:  "premium",
		ValidFrom: time.Now(),
		ValidTill: time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
	}
	// User subscribes to a plan.
	s := Subscription{
		ID:        "1",
		UserID:    u.ID,
		ValidFrom: time.Now(),
		ValidTill: time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
	}
	// The plan is created for the current month.
	sp := SubscriptionPlan{
		SubscriptionID: "1",
		PlanID:         "1",
		ValidFrom:      time.Now(),
		ValidTill:      time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
		IsPaid:         true,
	}
	sp1 := SubscriptionPlan{
		SubscriptionID: "1",
		PlanID:         "1",
		ParentID:       "1",
		ValidFrom:      time.Now(), // start of the month. NOTE: Check the period first, if it's annual, it should be start/end of the year, and the pricing deduction should be based on the difference.
		ValidTill:      time.Now(), // end of the month.
	}
	// To compute the final subscription values when the user upgrade/downgrade/terminate their plan:
	// Get all subscriptions where the valid_from is within the current period (month, year...).
	// Find the difference in days (plan 1 duration, plan 2 duration)
	// Compute the difference.
	// What if the user is attempting to modify the subscription frequently (?). Block them.
	fmt.Println("Hello, playground")
}
```


## Schema

```

party
- id
- subtype enum(person, organization)

// Subscription information for the user. If the plan is basic, it won’t be counted as a subscription to avoid creating redundant roles.
subscription
- id
- party_id
- valid_from // The date the subscription is activated
- valid_till // The date the subscription is expected to end (can be different than deleted at)
- is_active // The subscription status, or just check the date of valid_till
- created_at // The date the subscription is created.
- updated_at
- deleted_at


// Feature type
feature_type 
- name // E.g. country, period (weekly, monthly, yearly), currency
- description

// Feature represents the chosen feature type and it’s corresponding value.
feature
- id
- feature_type_id
- value

feature 
{id: 1, feature_type_id: currency, value: “SGD”},
{id: 2, feature_type_id: period, value: “monthly”}
{id: 3, feature_type_id: country, value: “Singapore”}
{id: 4, feature_type_id: max_clients, value: 20}

// Plan describes the value of the feature. Each plan will have a feature and a designated value. There are only three plans at most, but with different combination of features.
plan
- id
- name // The name of the plan (basic, elite, enterprise)
- description // The description of the plan.
- billing_method_type (auto, manual (?) better naming please)
- cost
- valid_from 
- valid_till // If we are going to deprecate a plan…

plan_feature
- plan_id
- feature_id
- cost
- duration_feature (yearly/monthly)

subscription plan
- subscription_id
- plan_id
- valid_from
- valid_till
- superseded_by (the previous subscription plan)
- // NOTE: This can be part of the feature.
- // country (subscription is different per country)
- // currency (currency is different per country)
- // cost (the cost depends on currency)
- // duration_feature (yearly/monthly)
- // is_renewable (?) can just check the valid_till date
- // status (?)


invoice 
- subscription_plan_id
- paid_amount (probably need this to offset the upgrade)
- amount
- for_date (what month/year is this invoice for?)
```
