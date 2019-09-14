# Spec

This expliains how this library works  

In the bases this library works has 3 types of packages that make it possible to send data over from one to another in a reliable way  
1. Data messages
2. Request messages
3. Confirm messages

### `1` Data messages
A data message contains data send by the end user that it want's to send over to the defined client.  
All data is send over multiple packges where each packges contains a bit of data.  
The message packges is made out of 3 parts where the first part is the message type id after that there is hte meta data for the message and at last the actual message data.  
The meta pacakge always has a fixed size of 41 bytes, within those bytes are these things stored:
```
Length  From-To  ValuType    Description
1       0-1      Bool (0/1)  start (1/0)
32      1-33     utf8        message id (a uuid without the "-")
4       33-37    uint        send from ([from % 255, from / (255^1) % 255, from / (255^2) % 255, from / (255^3) % 255])
4       37-41    uint        total message length ([length % 255, length / (255^1) % 255, length / (255^2) % 255, length / (255^3) % 255])
```
The message id is always 1 byte with a utf8 encoded `d`

The data contains the data from the `from` item in the meta data.  
The length of the data is dependent on the udp message size.  

These packages are expected to be send in a preditable from location.  
That means that if the data length is 1024 bytes and the current from meta item is 3072 we expect there to be a package with an from mata item of 4096 and not 10 bytes less or more.  

### `2` Request messages
A request messages is send by the reciver and contains a reqeust for missing parts.  
The message is expected to be send like this:
```
0x63 "r" > the message id
  > utf8 messageID wihout the "-"
0x00 > Null byte
  > uint request from
```

### `3` Confirm messages
A Confirm message is send by the reciver to confirm to a defined part of the message has ben recived.  
The message is expected to be send like this:
```
0x63 "c" > the message id
  > utf8 messageID wihout the "-"
0x72 Null byte
  > uint confirmed to
```
