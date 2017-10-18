namespace go net_interface

/*
struct Function
{
    //时间戳
    1: string timestamps;
    //计数
    2: string counts;
}

 typedef list<Function> FunctionList
*/

exception CountServiceException {
    1: i32 code
    2: string msg
}

service CountService {
    //i64 sendMessageToServer(1: string appid, 2: list<FunctionList> channels) throws (1: CountServiceException csException)
    i64 sendMessageToServer(1: string msg) throws (1: CountServiceException csException)
}