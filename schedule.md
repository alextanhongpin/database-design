# Scheduling data vs just in time

When dealing with mutable data, we can update them immediately so that changes are immediate, or we sometimes want to schedule the changes to take into affect in the future.


Lets take an example of publishing products. The simplest design is to add a published at timestamptz column for postgres. Note that this is more useful than a boolean flag because you can schedule them. By that, it means you can set published date to Jan 2021, and the code chcecks if now is greater than published at, then display.

what if we want the post to be unpublished after a certain duration. This could happen for ads or job posting, where we are paid to display them for a certain duration. In this case, we need two columns, or in postgres, we can just use tstzrange and check if now overlaps with the valid time.


the just in time approach is simpler, in a way that you just manually publish or unpublish data. However, it may become unmanageable when table size grows.


# Present and the future

Lets look at a more complex example, ranking. You are running a campaign, and you display them according to rank on your application. 

At the moment, it's campaign Alpha and Beta, ranked 1 and 2 immediately. You received a request to include a new campaign Charlie to be scheduled next week.

You know what to so, just include a mew tstzrange column to publish it. But your PM wants it to be scheduled at first rank when its published, and Alpha to be ranked last. The new order should be Charlie, Alpha and Beta.

Here is the hard part - the validity doesnt say anything about the ranking changes. Ranking is done regardless of whether the campaign is active or not now.

We can rank Charlie to be first now, it just wont be visible, but we cant alter the ranking of Alpha to be last now as it should only take effect next week.

There are a few options
1. create another table and schedule the ranking changes, much complexity
2. run a cron to schedule
3. manually update next week

Other things to note
1. if ranking is unique, you need to defer the constraints, otherwise bulk update will conflict
2. for simplicity, rank new campaigns first, which is min rank minus one
3. allow negative rank to cheat 
