## Sample golang code

```go
package main

import (
	"log"
	"time"
)

// type FeatureType string
type PlanFeature struct {
	FeatureID int
	Type      string
	Value     string
}

type Plan struct {
	ID        int
	ValidFrom time.Time
	ValidTill time.Time
	Cost      float64
	Name      string
	// The relationship should be inversed.
	// FeatureType FeatureType // Size, Country, etc.
	// The feature that it supersedes.
	ParentID int
}

func main() {
	/*
		Price of Coffee
		| id | country | size | currency | valid_from | valid_till | parent_id | price |
		| id | malaysia | small | MYR | 2019-01-01 | 9999-12-31 | - | 7 |
		| id | malaysia | medium | MYR | 2019-01-01 | 9999-12-31 | - | 8 |
		| id | malaysia | large | MYR | 2019-01-01 | 2019-01-02 | - | 9 |
		| id | malaysia | large | MYR | 2019-01-02 | 9999-12-31 | - | 10 |
	*/

	plans := []Plan{
		{
			ID:        1,
			Cost:      7,
			ValidTill: time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
		},
		{
			ID:        2,
			Cost:      9,
			ValidTill: time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
		},
		{
			ID:        3,
			Cost:      10,
			ValidTill: time.Now(),
		},
		{
			ID:        4,
			Cost:      11,
			ValidFrom: time.Now(),
			ValidTill: time.Date(9999, 12, 31, 0, 0, 0, 0, &time.Location{}),
			ParentID:  3,
		},
	}

	var planFeatures []PlanFeature
	planFeatures = []PlanFeature{
		{1, "size", "small"},
		{1, "country", "malaysia"},
	}
	// Select valid features.
	for _, p := range plans {
		valid := struct {
			start, end bool
		}{
			time.Now().Equal(p.ValidFrom) || time.Now().After(p.ValidFrom),
			time.Now().Before(p.ValidTill),
		}
		if valid.start && valid.end {
			log.Println(p)
		}
	}
	// Select features for the feature type 1
	for _, planFeature := range planFeatures {
		if planFeature.FeatureID == 1 {
			log.Println(planFeature)
		}
	}
}
```

Note: The design above only considers one value. What if the pricing is affected by multiple pricing?

## References:

- https://dba.stackexchange.com/questions/216317/pricing-table-mysql-table-design-i-need-the-database-design-for-the-the-plans
- https://softwareengineering.stackexchange.com/questions/307214/designing-pricing-table-rdbms-agnostic
- https://stackoverflow.com/questions/14546539/database-design-for-pricing-overview


## Unit

Store the unit amount instead in db, without numeric, and another column called currency. So if you wan to store MYR 1, the value will be 100 in the db. 

When communicating through API however, you can use MYR, but multiply by 100 on server side. This could be relevant if you are using multiple currency and want to aboid leaking business logic to the frontend. Stripe however, enforces the client to use unit amount, so user have to send 100 cents instead of MYR 100. This makes sense, since in some cases, payments can be down to several tenth of cents (e.g mobile data usage etc). In that scenario, Stripe has a different convention, which is unit decimal amount. 

So if we allow client to send float, RM 1.4509 vs 145.09 unit amount in cents, we do not know if users explicitly wants to send cents with decimal or not. There will be pain points in converting on client when working with multiple currencies though where you need to know how much to multiply with. 

https://stripe.com/docs/billing/subscriptions/decimal-amounts


