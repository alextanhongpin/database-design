
http://www.silota.com/docs/recipes/

## Some thoughts on 

Some recipes for sql analytics

## Smoothing data
Raw data can be noisy. To reduce noise, we perform smoothing on the data before presenting it.

- calculating running total
- calculating running/moving average
- calculating weighted moving average
- calculating exponential moving average

## Calculations per group

Key business decisions are made based on comparison across various product, customer and employee groups. Typical use of window function is to calculate sales compensation plans and quotas, year-to-date comparisons and hot selling products across categories.

- calculating percentage of total sum
- calculating differences from beginning/first row
- calculating top n items per group
- calculating top n items and aggregating (sum) the remainder into all other
- calculating distinct and unique items per group

## Growth rates

We can measure growth rates by modelling month-over-month and exponential rates and pareto charts in order to compare and focus your efforts.

- calculating month-over-month growth rate
- calculating exponential growth rate
- creating pareto charts to visualize the 80/20 principles

## Summarizing data

To get a birds-eye-view of the data, we can look at the shape of the data, bucketing into groups, finding outliers, calculating relationships and correlations.

- calculating summaries and descriptive statistics
- calculating summaries with histogram and frequency distributions
- calculating relationships with correlation matrices
- calculating n-tiles (quartiles, deciles and percentiles)
- calculating z-scores
- gap analysis to find missing values in a sequence

## Ranking your best and worst customers

Understanding your best and worst customers is key to profitable growth. Use lead-scoring and net promoter score surveys to rank your customers.

- Analysing Recency, Frequency, and Monetary (RFM) value to index your best customers
- segmenting and lead scoring your email list
- analysing net promoter score (NPS) surveys to improve customer satisfaction and loyalty


## Forecasting and predicting the future

Accurate forecasting of future activity is useful when provisioning resources and maintaining sufficient lead time.

- calculating linear regressions coefficients
- forecasting in presence of seasonal effects using the ratio to moving average methods

## SQL for marketing

Marketing team can utilise sql to understand return on ad spend, attribute revenue to marketing programs and payback periods of different marketing channels.

- multichannel marketing attribution modeling
- funnel analysis

## Data cleansing (aka Wrangling)

Dirty data can lead you astray. Understand pattern matching (e.g. business emails) filling missing data, removing duplicates and empty values to sufficiently deal with messy data.

- finding duplicate rows
- filling missing data and plugging gaps by generating a continuous series
- finding patterns and matching substrings using regular expressions
- concatenating rows of string values for aggregation
- sql null values - comparing, sorting, converting and joining with real values

## Business model analytics
- using sql to measure stocks performance
- estimating demand curves and profit maximizing pricing
- account-level CRM analytics for B2B SaaS Companies

## Others

- comparing means with statistical testing
- calculating medians
- calculating fractional and ordinal rank
- calculating n-grams
- calculating funnel drop-off metrics
- cohort charts for retention analysis
- understanding explain analyse
- pivoting and unpivoting data
- intrusion detection with ip addresses (how to create your own ip blacklist)
- GIS/spatial queries
- correlated and uncorrelated subqueries


## SQL for machine learning

- linear regression
- decision tree
- finding nearest neighbour
- clustering documents
- finding levenshtein distance
- autocorrect
- locality sensitive hashing
- finding frequent itemsets in the database (market basket analysis)

## SQL for rule engine

- business rule engine in the database
