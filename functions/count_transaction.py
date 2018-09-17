def count_transaction(request):
    from google.cloud import datastore
    client = datastore.Client()
    with client.transaction():
        my_key = client.key("TransactionCounter", "AllPurchases")
        c = client.get(my_key) 
        if c is None:
            print("counter not found, inserting new")
            c = datastore.Entity(key=my_key)
            c['count'] = 1 
            client.put(c)
        else:
            print("counter found! incrementing. %s" % c)
            c['count'] = c['count'] + 1 
        client.put(c)



