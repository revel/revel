/*
sessvars ver 1.01
- JavaScript based session object
copyright 2008 Thomas Frank

This EULA grants you the following rights:

Installation and Use. You may install and use an unlimited number of copies of the SOFTWARE PRODUCT.

Reproduction and Distribution. You may reproduce and distribute an unlimited number of copies of the SOFTWARE PRODUCT either in whole or in part; each copy should include all copyright and trademark notices, and shall be accompanied by a copy of this EULA. Copies of the SOFTWARE PRODUCT may be distributed as a standalone product or included with your own product.

Commercial Use. You may sell for profit and freely distribute scripts and/or compiled scripts that were created with the SOFTWARE PRODUCT.

v 1.0 --> 1.01
sanitizer added to toObject-method & includeFunctions flag now defaults to false

*/

sessvars=function(){

	var x={};
	
	x.$={
		prefs:{
			memLimit:2000,
			autoFlush:true,
			crossDomain:false,
			includeProtos:false,
			includeFunctions:false
		},
		parent:x,
		clearMem:function(){
			for(var i in this.parent){if(i!="$"){this.parent[i]=undefined}};
			this.flush();
		},
		usedMem:function(){
			x={};
			return Math.round(this.flush(x)/1024);
		},
		usedMemPercent:function(){
			return Math.round(this.usedMem()/this.prefs.memLimit);
		},
		flush:function(x){
			var y,o={},j=this.$$;
			x=x||top;
			for(var i in this.parent){o[i]=this.parent[i]};
			o.$=this.prefs;
			j.includeProtos=this.prefs.includeProtos;
			j.includeFunctions=this.prefs.includeFunctions;
			y=this.$$.make(o);
			if(x!=top){return y.length};
			if(y.length/1024>this.prefs.memLimit){return false}
			x.name=y;
			return true;
		},
		getDomain:function(){
				var l=location.href
				l=l.split("///").join("//");
				l=l.substring(l.indexOf("://")+3).split("/")[0];
				while(l.split(".").length>2){l=l.substring(l.indexOf(".")+1)};
				return l
		},
		debug:function(t){
			var t=t||this,a=arguments.callee;
			if(!document.body){setTimeout(function(){a(t)},200);return};
			t.flush();
			var d=document.getElementById("sessvarsDebugDiv");
			if(!d){d=document.createElement("div");document.body.insertBefore(d,document.body.firstChild)};
			d.id="sessvarsDebugDiv";
			d.innerHTML='<div style="line-height:20px;padding:5px;font-size:11px;font-family:Verdana,Arial,Helvetica;'+
						'z-index:10000;background:#FFFFCC;border: 1px solid #333;margin-bottom:12px">'+
						'<b style="font-family:Trebuchet MS;font-size:20px">sessvars.js - debug info:</b><br/><br/>'+
						'Memory usage: '+t.usedMem()+' Kb ('+t.usedMemPercent()+'%)&nbsp;&nbsp;&nbsp;'+
						'<span style="cursor:pointer"><b>[Clear memory]</b></span><br/>'+
						top.name.split('\n').join('<br/>')+'</div>';
			d.getElementsByTagName('span')[0].onclick=function(){t.clearMem();location.reload()}
		},
		init:function(){
			var o={}, t=this;
			try {o=this.$$.toObject(top.name)} catch(e){o={}};
			this.prefs=o.$||t.prefs;
			if(this.prefs.crossDomain || this.prefs.currentDomain==this.getDomain()){
				for(var i in o){this.parent[i]=o[i]};
			}
			else {
				this.prefs.currentDomain=this.getDomain();
			};
			this.parent.$=t;
			t.flush();
			var f=function(){if(t.prefs.autoFlush){t.flush()}};
			if(window["addEventListener"]){addEventListener("unload",f,false)}
			else if(window["attachEvent"]){window.attachEvent("onunload",f)}
			else {this.prefs.autoFlush=false};
		}
	};
	
	x.$.$$={
		compactOutput:false, 		
		includeProtos:false, 	
		includeFunctions: false,
		detectCirculars:true,
		restoreCirculars:true,
		make:function(arg,restore) {
			this.restore=restore;
			this.mem=[];this.pathMem=[];
			return this.toJsonStringArray(arg).join('');
		},
		toObject:function(x){
			if(!this.cleaner){
				try{this.cleaner=new RegExp('^("(\\\\.|[^"\\\\\\n\\r])*?"|[,:{}\\[\\]0-9.\\-+Eaeflnr-u \\n\\r\\t])+?$')}
				catch(a){this.cleaner=/^(true|false|null|\[.*\]|\{.*\}|".*"|\d+|\d+\.\d+)$/}
			};
			if(!this.cleaner.test(x)){return {}};
			eval("this.myObj="+x);
			if(!this.restoreCirculars || !alert){return this.myObj};
			if(this.includeFunctions){
				var x=this.myObj;
				for(var i in x){if(typeof x[i]=="string" && !x[i].indexOf("JSONincludedFunc:")){
					x[i]=x[i].substring(17);
					eval("x[i]="+x[i])
				}}
			};
			this.restoreCode=[];
			this.make(this.myObj,true);
			var r=this.restoreCode.join(";")+";";
			eval('r=r.replace(/\\W([0-9]{1,})(\\W)/g,"[$1]$2").replace(/\\.\\;/g,";")');
			eval(r);
			return this.myObj
		},
		toJsonStringArray:function(arg, out) {
			if(!out){this.path=[]};
			out = out || [];
			var u; // undefined
			switch (typeof arg) {
			case 'object':
				this.lastObj=arg;
				if(this.detectCirculars){
					var m=this.mem; var n=this.pathMem;
					for(var i=0;i<m.length;i++){
						if(arg===m[i]){
							out.push('"JSONcircRef:'+n[i]+'"');return out
						}
					};
					m.push(arg); n.push(this.path.join("."));
				};
				if (arg) {
					if (arg.constructor == Array) {
						out.push('[');
						for (var i = 0; i < arg.length; ++i) {
							this.path.push(i);
							if (i > 0)
								out.push(',\n');
							this.toJsonStringArray(arg[i], out);
							this.path.pop();
						}
						out.push(']');
						return out;
					} else if (typeof arg.toString != 'undefined') {
						out.push('{');
						var first = true;
						for (var i in arg) {
							if(!this.includeProtos && arg[i]===arg.constructor.prototype[i]){continue};
							this.path.push(i);
							var curr = out.length; 
							if (!first)
								out.push(this.compactOutput?',':',\n');
							this.toJsonStringArray(i, out);
							out.push(':');                    
							this.toJsonStringArray(arg[i], out);
							if (out[out.length - 1] == u)
								out.splice(curr, out.length - curr);
							else
								first = false;
							this.path.pop();
						}
						out.push('}');
						return out;
					}
					return out;
				}
				out.push('null');
				return out;
			case 'unknown':
			case 'undefined':
			case 'function':
				if(!this.includeFunctions){out.push(u);return out};
				arg="JSONincludedFunc:"+arg;
				out.push('"');
				var a=['\n','\\n','\r','\\r','"','\\"'];
				arg+=""; for(var i=0;i<6;i+=2){arg=arg.split(a[i]).join(a[i+1])};
				out.push(arg);
				out.push('"');
				return out;
			case 'string':
				if(this.restore && arg.indexOf("JSONcircRef:")==0){
					this.restoreCode.push('this.myObj.'+this.path.join(".")+"="+arg.split("JSONcircRef:").join("this.myObj."));
				};
				out.push('"');
				var a=['\n','\\n','\r','\\r','"','\\"'];
				arg+=""; for(var i=0;i<6;i+=2){arg=arg.split(a[i]).join(a[i+1])};
				out.push(arg);
				out.push('"');
				return out;
			default:
				out.push(String(arg));
				return out;
			}
		}
	};
	
	x.$.init();
	return x;
}()


