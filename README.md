> [!WARNING]  
> This idea/fork is still at early works. It is technically working - you just have to provide proper Gateway Packets handler. There are no promises regarding it's availability - please fork this side lib code if you wish to use it long term.

### Alternative lib: Qord

I've forked Tempest's master branch with idea of having alternative client that uses discord gateway instead http for people who wants very low latency and don't care about having to keep connections alive (they host bot on their own VPS instead cloud, etc.).

### Ideas over master branch
- Use gateway to halve latency,
- Instantly update bot's avatar & banner images thanks to constant heartbeat

### Why?

Discord API / Webhook services were in last weeks rather unstable. Various bot owners noticed random lag spikes that were so big that it makes some bot hardly usable. This lib is a barebone implementation of what Tempest could be if it used Gateway session.

The code has been moved into own packages to make maintenaining it bit easier - this however will require some changes to the code paths if you plan on switching Tempest App into Qord Bot. I'll try my best to keep compatibility with how Tempest operates.