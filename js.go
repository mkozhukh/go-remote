package remote

import (
	"io"
	"text/template"
)

var apiRawText = `
(function(){

var scs = document.getElementsByTagName("script");
var url = scs[scs.length-1].getAttribute("src");
scs = null;



(function (root) {
	if (typeof exports === 'object') {
        // Node, CommonJS-like
        module.exports = Remote;
    } else {
        // Browser globals (root is window)
       	root.remote = Remote;
    }
}(this));


function Remote(config){
	if (typeof config === "string")	{
		return fetchConfig(config).then(function(cfg){
			return cfg.json();
		}).then(function(cfg){
			return Remote(cfg).active;
		});
	}

	var master = { version: config.version };

	var url = config.url;
	var csrf = config.key;

	var queue = [];
	var seed = 1;
	var socket = null;
	

	master.data = {};
	for (var key in config.data)
		master.data[key] = config.data[key];
	
	master.api = {};
	for (var key in config.api){
		var obj = {};
		var cfg = config.api[key];
		for (var method in cfg)
			obj[method] = wrapper(key+"."+method);
		master.api[key] = obj;
	}

	if (config.websocket && window.WebSocket){
		master.active = new Promise(function(res){
			const sbs = {};
			const and = config.websocket.indexOf("?")!=-1?"&":"?";
			socket = new WebSocket(config.websocket+and+"key="+csrf);
			socket.addEventListener('message', function (ev) {
				const pack = JSON.parse(ev.data);
				if (pack.action === "result")
					parseData(pack.data, []);
				else if (pack.action === "event")
					triggerEvent(pack.data)
				else if (pack.action == "active")
						res(master);
				else {
					if (master.onerror)
						master.onerror(pack.data || "WebSocket error");
				}
			});

			function triggerEvent(data){
				const all = sbs[data.name];

				if (all){
					for (var i=0; i<all.length; i++)
						all[i].handler(data.data);
				}
			}
			master.attachEvent = function(name, handler){
				const id = uid();
				const all = sbs[name] = sbs[name] || [];
				all.push({ id:id, handler:handler });

				socket.send(JSON.stringify({ action:"subscribe", name }));
				return { name,id };
			};
			master.detachEvent = function(pack){
				let all = sbs[pack.name];
				if (all) all = sbs[pack.name] = all.filter(a => a.id != pack.id);
				if (!all.lenght){
					delete sbs[pack.id];
					socket.send(JSON.stringify({ action:"unsubscribe", name }));
				}
			};
			master.on = { attachEvent : master.attachEvent, detachEvent:master.detachEvent };
		});
	} else {
		master.active = Promise.resolve(master);
	}


	
	function uid(){
		return (seed++).toString();
	}
	
	function wrapper(key){
		return function(){
			var args = [].slice.call(arguments);
			return new Promise(function(resolve, reject){
				queue.push({
					data:{
						id:uid(),
						name:key,
						args: args
					},
					status:"new",
					resolve:resolve,
					reject:reject
				});
				setTimeout(send, 1);
			});
		};
	}
	
	function send(name, args){
		var pack = queue.filter(function(obj){ return obj.status === "new"; }).map(function(obj){
			obj.status = "wait";
			return obj.data
		});

		if (!pack.length) return;
		if (socket){
			return socket.send(JSON.stringify({ action:"remote", body:pack }));
		}
		
		var headers = {
			'Accept': 'application/json',
			'Content-Type': 'application/json',
			"Remote-CSRF": csrf
		};
		var data = window.fetch ? 
		fetch(url, {
			method: "POST",
			credentials: "include",
			headers:headers,
			body:JSON.stringify(pack)
		}) 
		: 
		webix.ajax().headers(headers).post(url, JSON.stringify(pack)).then(function(obj){ parseData(obj.json(), pack); });

		data["catch"](function(){ return false; }).then(function(res){ 
			if (res && res.ok){
				res.json().then(function(data){ parseData(data, pack); });
			} else {
				parseData(false, pack);
			}
		});

		if (master.onload){
			master.onload(data);
		}
	}

	function fetchConfig(url){
		var headers = { 'Accept': 'application/json' };

		return window.fetch ?
			fetch(url, { credentials: "include", headers:headers})
			:
			webix.ajax().headers(headers).get(url);
	}

	function parseData(data, pack){
		var all = {};

		if (!data){
			for (var i=0; i<pack.length; i++)
				all[pack[i].id] = { error:"Network Error" };
		} else {
			for (var i=0; i<data.length; i++)
				all[data[i].id] = data[i];
		}

		result(queue, all);
	}
	
	function result(queue, all){
		for (var i=queue.length-1; i>=0; i--){
			var test = queue[i];
			var check = all[test.data.id];
			if (check){
				if (check.error){
					test.reject(check.error);
					if (master.onerror)
						master.onerror(check.error);
				} else
					test.resolve(check.data);

				queue.splice(i, 1);
			}
		}
	}

	return master;
}


var config = {{.Config}};
config.url = config.url || url;
window.{{.Name}} = Remote(config);
}).call({});

`


var tmpl, _ = template.New("api").Parse(apiRawText)

type apiVars struct {
	Name   string
	Config string
}

func apiText(w io.Writer, name string, api string) {
	err := tmpl.Execute(w, &apiVars{name, api})
	if err != nil {
		log.Errorf(err.Error())
	}
}
