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
