# protoc-gen-go-orm

## Getting started

## Description

### CAS

* mysql: 通过声明 int64 version = x; 声明使用乐观锁，每次更新时，version 字段会自动比对并加 1
* tcaplus: 通过声明 int64 version = x; 声明使用乐观锁。实际没有使用此字段做任何操作，仅作为标记使用，通过tcaplus自身的version机制实现乐观锁

### tcaplus

`tcaplus` 表结构要求：

- 主键必须是单字段主键
- 分片键必须是主键的一部分
- 索引字段必须是主键的一部分
- 分片键必须在索引的最前面

`tcaplus` 文件要求：

- 必须引用`[tcaplusservice.optionv1.proto](options/tcaplus/tcaplusservice.optionv1.proto)`；
- proto文件创建后需要上传，此后不可修改，任何修改都需要重新上传；


`tcaplus`索引的用途：

- `一对多`的场景用部分主键查询，例如：通过 user_id 查询该用户的订单，user_id和order_id 都是主键字段
- 如果有非主键字段索引查询的需求，最好额外建立一个索引表。
例如：通过 email 查询用户信息，可以建立一个 email_user 表，email 作为主键，user_id 作为普通字段。通过数据冗余来满足查询需求

## Installation

## Usage

## Roadmap

- [ ] fatal in template
- [ ] embed of mysql
- [ ] redis driver support
- [ ] custom tags support
- [ ] more examples
