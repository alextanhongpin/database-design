# How to deal with data changes on the client side

## Example 1: Ecommerce payment

During payment, the prices of some products could have changed. Normally, once the user decided to checkout, an invoice will be created to record the price of the transaction. However, the price could still have changed before the invoice is created. 

One way to prevent this is to request the user to send the payment information to the server to validate and compare against the current price. Then an appropriate error message can be shown to the user if there is a mismatch.

## Example 2: Ecommerce checkout order out of stock

Sometimes the product can be out of stock when the user already added the product in the cart. Before the user checkout, we can refetch the stocks data again. 

If the product went out of stock after that, we can still validate it on the server side.
