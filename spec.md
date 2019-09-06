# Spec

This expliains how this library works  

In the bases this library works has 3 types of packages that make it possible to send data over from one to another in a reliable way  
1. Data messages
2. Request messages
3. Confirm messages

### `1` Data messages
A data message contains data send by the end user that it want's to send over to the defined client.  

### `2` Request messages
A request messages is send by the reciver and contains a reqeust for missing parts.  

### `3` Confirm messages
A Confirm message is send by the reciver to confirm to a defined part of the message has ben recived
