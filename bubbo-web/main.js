/**
 * Created by binoz on 2017/3/12.
 */
"use strict";

class MyEvent {
  constructor(Type,Detail,Group) {
    this.Group =Group || '*';
    this.Type = Type||'';
    this.Detail = Detail||{};
  }

  path(){
    return '/'+this.Group+'/'+this.Type;
  }
}

function EventFromJSON(event){
  return new MyEvent(event.Type,event.Detail,event.Group);
}


