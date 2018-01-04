package remote

import (
	"io"
	"text/template"
)

var apiRawText = `(function(master, config){

	var scs = document.getElementsByTagName("script");
	var url = scs[scs.length-1].getAttribute("src");
	scs = null;

	var queue = [];
	var seed = 1;
	var csrf = config.key;
	
	master.data = master.data||{};
	master.api = master.api||{};
	master.version = config.version;

	for (var key in config.data)
		master.data[key] = config.data[key];
	for (var key in config.api){
		var obj = {};
		var cfg = config.api[key];
		for (var method in cfg)
			obj[method] = wrapper(key+"."+method);
		master.api[key] = obj;
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
		var pack = queue.filter(obj => obj.status === "new").map(obj => {
			obj.status = "wait";
			return obj.data
		});

		if (!pack.length) return;
	
		fetch(url, {
			method: "POST",
			credentials: "include",
			headers:{
				'Accept': 'application/json',
				'Content-Type': 'application/json',
				"Remote-CSRF": csrf
			},
			body:JSON.stringify(pack)
		}).catch( _ => false).then(res => {
			var all = {};
	
			if (!res || !res.ok){
				for (var i=0; i<pack.length; i++)
					all[pack[i].id] = { error:"Network Error" };
					result(queue, all)
			} else {
				res.json().then(function(data){
					for (var i=0; i<data.length; i++)
						all[data[i].id] = data[i];
					result(queue, all)
				});
			}

		})
	}
	
	function result(queue, all){
		for (var i=queue.length-1; i>=0; i--){
			var test = queue[i];
			var check = all[test.data.id];
			if (check){
				if (check.error)
					test.reject(check.error);
				else
					test.resolve(check.data);

				queue.splice(i, 1);
			}
		}
	}
})((window.{{.Name}} = window.{{.Name}} || {}), {{.Config}})`

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
