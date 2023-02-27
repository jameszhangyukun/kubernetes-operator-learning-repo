### client-go 架构
 <div align=center>
<img src="./image/client-go-controller-interaction.jpeg"> 
</div>

1. Reflector通过List/Watch机制获取到ApiServer中数据对象的变化
2. 将数据变化的对象添加到Delta队列中
3. Informer冲队列中取出数据
4. 将数据添加到Indexer中
5. 将变化事件推送到Controller的Worker中，由Worker进行处理

