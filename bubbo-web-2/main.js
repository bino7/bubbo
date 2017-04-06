/**
 * Created by binoz on 2017/3/12.
 */
"use strict";

class MyEvent {
  constructor(Type,Detail,Group,Id) {
    if (!Id) {
      var uuid = require('uuid');
      Id = uuid.v4();
    } 
    this.Group =Group || '*';
    this.Type = Type||'';
    this.Detail = Detail||{};
  }

  path(){
    return '/'+this.Group+'/'+this.Type;
  }
}

function EventFromJSON(event){
  return new MyEvent(event.Type,event.Detail,event.Group,event.Id);
}


