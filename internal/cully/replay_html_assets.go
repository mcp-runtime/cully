package cully

// Inline assets for the offline replay page. They contain no external URLs and
// build every piece of recorded text with textContent, never innerHTML.

const replayHTMLStyle = `
:root{--bg:#fbfaf7;--fg:#1f2430;--dim:#6b7280;--card:#ffffff;--line:#d9d6cc;--accent:#2f6fdd;--ok:#1f8f4e;--bad:#cf2f3d;--warn:#b7791f;--explore:#12808a;--impl:#2f6fdd;--verify:#1f8f4e;--fix:#b7791f;--mem:#8a3fb0;--run:#6b7280;--hot:#ffb020}
@media (prefers-color-scheme:dark){:root:not([data-theme="light"]){--bg:#14161c;--fg:#e6e8ee;--dim:#8b93a3;--card:#1c2029;--line:#2c3240;--accent:#6ea0ff;--ok:#4cc38a;--bad:#ff6b78;--warn:#f0b650;--explore:#4fc3cd;--impl:#6ea0ff;--verify:#4cc38a;--fix:#f0b650;--mem:#c48bf0;--run:#8b93a3;--hot:#ffb020}}
:root[data-theme="dark"]{--bg:#14161c;--fg:#e6e8ee;--dim:#8b93a3;--card:#1c2029;--line:#2c3240;--accent:#6ea0ff;--ok:#4cc38a;--bad:#ff6b78;--warn:#f0b650;--explore:#4fc3cd;--impl:#6ea0ff;--verify:#4cc38a;--fix:#f0b650;--mem:#c48bf0;--run:#8b93a3;--hot:#ffb020}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);font:14px/1.45 ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif}
code,.mono{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:12.5px}
header{padding:18px 20px 6px}
h1{font-size:20px;margin:0 0 2px}
.sub{color:var(--dim)}
.bar{position:sticky;top:0;z-index:5;background:var(--bg);border-bottom:1px solid var(--line);padding:10px 20px}
.controls{display:flex;flex-wrap:wrap;gap:8px;align-items:center;margin-bottom:8px}
button,select,input[type=text]{font:inherit;color:var(--fg);background:var(--card);border:1px solid var(--line);border-radius:6px;padding:4px 10px}
button{cursor:pointer}
button:focus-visible,select:focus-visible,input:focus-visible,.node:focus-visible{outline:2px solid var(--accent);outline-offset:1px}
.scrub{position:relative;height:34px}
.markers{position:absolute;left:0;right:0;top:0;height:14px}
.markers span{position:absolute;transform:translateX(-50%);font-size:12px;line-height:14px}
.markers .loop{color:var(--warn)}.markers .fail{color:var(--bad)}
.scrub input[type=range]{position:absolute;left:0;right:0;bottom:0;width:100%;margin:0;accent-color:var(--accent)}
.times{display:flex;justify-content:space-between;color:var(--dim);font-size:12px}
main{display:grid;grid-template-columns:minmax(0,1.1fr) minmax(0,1fr);gap:16px;padding:16px 20px}
@media (max-width:900px){main{grid-template-columns:1fr}}
section{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:12px 14px;min-width:0}
section h2{font-size:12px;letter-spacing:.08em;text-transform:uppercase;color:var(--dim);margin:0 0 8px}
.wide{grid-column:1/-1}
.node{border:1px solid var(--line);border-left:5px solid var(--run);border-radius:8px;padding:8px 10px;margin:0 0 4px;cursor:pointer;background:var(--bg)}
.node.k-Explore{border-left-color:var(--explore)}.node.k-Implement{border-left-color:var(--impl)}.node.k-Verify{border-left-color:var(--verify)}.node.k-Fix{border-left-color:var(--fix)}.node.k-Remember{border-left-color:var(--mem)}
.node.s-failed{border-left-color:var(--bad)}.node.s-mixed{border-left-color:var(--warn)}
.node.cur{outline:2px solid var(--accent);background:var(--card)}
.node .head{display:flex;justify-content:space-between;gap:8px;font-weight:600}
.node .meta{color:var(--dim);font-weight:400}
.node .body{margin-top:4px;color:var(--dim)}
.node .more{display:none;margin-top:6px;padding-top:6px;border-top:1px dashed var(--line)}
.node.open .more{display:block}
.arrow{text-align:center;color:var(--dim);line-height:1;margin:0 0 4px}
.retry{color:var(--warn);font-size:12px;margin-top:4px}
.warn{color:var(--warn)}.bad{color:var(--bad)}.ok{color:var(--ok)}.dim{color:var(--dim)}
#feed{max-height:420px;overflow:auto}
.step{display:grid;grid-template-columns:48px 70px minmax(0,1fr);gap:8px;padding:2px 6px;border-radius:5px;color:var(--dim)}
.step.cur{color:var(--fg);background:var(--bg);outline:1px solid var(--line)}
.step .d{overflow-wrap:anywhere}
.tree{max-height:420px;overflow:auto}
.dir{color:var(--dim);margin-top:6px;overflow-wrap:anywhere}
.file{display:grid;grid-template-columns:18px minmax(0,1fr) 110px;gap:6px;align-items:center;padding-left:14px}
.file .n{overflow-wrap:anywhere}
.heat{height:8px;border-radius:4px;background:var(--line);overflow:hidden}
.heat i{display:block;height:100%;background:var(--accent)}
.file.hot .heat i{background:var(--hot);box-shadow:0 0 8px var(--hot)}
.file.hot .n{font-weight:700}
.file.deleted .n{color:var(--bad);text-decoration:line-through}
.file.created .n{color:var(--ok)}
.foot{display:flex;flex-wrap:wrap;gap:14px;color:var(--dim);margin-top:8px}
ul.plain{margin:4px 0 0 18px;padding:0}
footer{padding:4px 20px 28px;color:var(--dim);font-size:12px}
`

const replayHTMLScript = `
(function(){
"use strict";
var D=JSON.parse(document.getElementById("data").textContent);
var steps=D.steps;
var start=Date.parse(D.meta.start),end=Date.parse(D.meta.end);
var BADGE={create:"+",edit:"~",write:"~","delete":"−",move:"→",read:"·"};
var pos=steps.length,playing=false,speed=1,timer=null;
var filt={op:"",file:"",agent:""};
var open={};
function $(id){return document.getElementById(id);}
function el(tag,cls,text){var e=document.createElement(tag);if(cls)e.className=cls;if(text!==undefined)e.textContent=String(text);return e;}
function clear(n){while(n.firstChild)n.removeChild(n.firstChild);}
function hm(t){return new Date(t).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"});}
function dur(s){if(s<60)return s+"s";if(s<3600)return Math.floor(s/60)+"m";return Math.floor(s/3600)+"h"+("0"+(Math.floor(s/60)%60)).slice(-2)+"m";}
function frac(t){var tot=end-start;if(tot<=0)return 0;return Math.min(1,Math.max(0,(t-start)/tot));}
function visible(s){
  if(filt.agent&&filt.agent!==D.agent)return false;
  if(filt.op){
    var ok=(s.op===filt.op)||(filt.op==="edit"&&s.op==="write")||(filt.op==="check"&&s.kind==="check")||(filt.op==="run"&&s.kind==="run");
    if(!ok)return false;
  }
  if(filt.file){
    if(s.kind!=="file")return false;
    if(String(s.path).indexOf(filt.file)<0&&String(s.to||"").indexOf(filt.file)<0)return false;
  }
  return true;
}
function curTime(){return pos>0?Date.parse(steps[pos-1].t):start;}
function renderFeed(){
  var root=$("feed");clear(root);var last=null;
  for(var i=0;i<pos;i++){
    var s=steps[i];if(!visible(s))continue;
    var row=el("div","step"+(i===pos-1?" cur":""));
    row.appendChild(el("span","mono",hm(Date.parse(s.t))));
    row.appendChild(el("span","",s.kind==="session"||s.kind==="note"||s.kind==="loop"?"":s.verb));
    var d=el("span","d",s.detail||"");
    if(s.outcome==="failed"){d.appendChild(el("span","bad"," ✕ failed"));}
    if(s.outcome==="passed"){d.appendChild(el("span","ok"," ✓ passed"));}
    if(s.kind==="loop")d.className+=" warn";
    row.appendChild(d);root.appendChild(row);last=row;
  }
  if(!last)root.appendChild(el("div","dim","Nothing to show yet."));
  else root.scrollTop=root.scrollHeight;
}
function renderFiles(){
  var root=$("files");clear(root);
  var st={},order=[];
  function get(p){if(!st[p]){st[p]={p:p,edits:0,reads:0,created:false,deleted:false,moved:false};order.push(p);}return st[p];}
  for(var i=0;i<pos;i++){
    var s=steps[i];
    if(s.kind!=="file"||s.failed)continue;
    if(filt.agent&&filt.agent!==D.agent)continue;
    if(filt.op&&!visible(s))continue;
    if(filt.file&&String(s.path).indexOf(filt.file)<0&&String(s.to||"").indexOf(filt.file)<0)continue;
    var f=get(s.path);
    if(s.op==="read")f.reads++;
    else if(s.op==="create"){f.created=true;f.deleted=false;}
    else if(s.op==="edit"||s.op==="write"){f.edits++;f.deleted=false;}
    else if(s.op==="delete")f.deleted=true;
    else if(s.op==="move"){f.deleted=true;if(s.to){var t=get(s.to);t.moved=true;t.deleted=false;}}
  }
  order.sort();
  if(!order.length){root.appendChild(el("div","dim","No files yet."));return;}
  var max=1;order.forEach(function(p){if(st[p].edits>max)max=st[p].edits;});
  var lastDir="\u0000";
  order.forEach(function(p){
    var f=st[p],i=p.lastIndexOf("/"),dir=i>=0?p.slice(0,i+1):"",name=i>=0?p.slice(i+1):p;
    if(dir!==lastDir){lastDir=dir;if(dir)root.appendChild(el("div","dir mono",dir));}
    var hot=f.edits>=3;
    var row=el("div","file"+(hot?" hot":"")+(f.deleted?" deleted":"")+(f.created?" created":""));
    var b=f.deleted?"−":f.created?"+":f.edits?"~":f.moved?"→":"·";
    row.appendChild(el("span","mono",b));
    row.appendChild(el("span","n mono",name+(f.edits?" ×"+f.edits:"")+(hot?" ▲":"")));
    var heat=el("span","heat"),bar=el("i");bar.style.width=(f.edits?Math.max(8,Math.round(f.edits/max*100)):0)+"%";
    heat.appendChild(bar);row.appendChild(heat);root.appendChild(row);
  });
}
function renderBar(){
  $("pos").max=String(steps.length);$("pos").value=String(pos);
  var f=pos>0?steps[pos-1].footer:{files:0,checks_passed:0,checks_failed:0,loops:0};
  var t=curTime();
  $("counts").textContent="files "+f.files+" · checks "+f.checks_passed+" passed / "+f.checks_failed+" failed · loops "+f.loops+" · "+dur(Math.round((t-start)/1000))+" of "+dur(Math.round((end-start)/1000));
  $("play").textContent=playing?"Pause":(pos>=steps.length?"Replay":"Play");
}
function render(){renderBar();renderFeed();renderFiles();}
function tick(){
  if(!playing)return;
  if(pos>=steps.length){playing=false;render();return;}
  pos++;render();
  var gap=pos<steps.length?Date.parse(steps[pos].t)-Date.parse(steps[pos-1].t):0;
  timer=setTimeout(tick,Math.max(80,Math.min(gap,1500))/speed);
}
function setPlaying(p){
  playing=p;clearTimeout(timer);
  if(p){if(pos>=steps.length)pos=0;render();timer=setTimeout(tick,300/speed);}else render();
}
function buildMarkers(){
  var root=$("markers");clear(root);
  steps.forEach(function(s){
    var kind=s.kind==="loop"?"loop":(s.kind==="check"&&s.failed)?"fail":"";
    if(!kind)return;
    var m=el("span",kind,kind==="loop"?"⚠":"✕");
    m.style.left=(frac(Date.parse(s.t))*100)+"%";
    m.title=kind==="loop"?"loop":"failed check";
    root.appendChild(m);
  });
}
function buildSummary(){
  var S=D.summary,root=$("summary");clear(root);
  var line=el("div","","Duration "+dur(S.duration_seconds)+" · checks "+S.checks_passed+" passed / "+S.checks_failed+" failed · loops "+S.loops);
  root.appendChild(line);
  var F=S.files;
  [["Created","+",F.created],["Edited","~",F.edited],["Deleted","−",F.deleted],["Moved","→",F.moved],["Read only","·",F.read_only]].forEach(function(g){
    if(!g[2]||!g[2].length)return;
    root.appendChild(el("div","dim",g[0]+" ("+g[2].length+")"));
    var ul=el("ul","plain mono");
    g[2].forEach(function(e){ul.appendChild(el("li","",g[1]+" "+e.path+(e.to?" → "+e.to:"")+(e.count>1?" ×"+e.count:"")));});
    root.appendChild(ul);
  });
  if(!F.recorded)root.appendChild(el("div","dim","No file activity was recorded for this session."));
  if(F.hotspots&&F.hotspots.length)root.appendChild(el("div","warn","Hotspots (edited 3+ times): "+F.hotspots.map(function(h){return h.path+" ×"+h.count;}).join(", ")));
  var R=D.reconcile,rr=$("reconcile");clear(rr);
  if(!R.available)rr.appendChild(el("div","dim","Git status was not available, so the journal could not be compared with the working tree."));
  else if(!R.comparable)rr.appendChild(el("div","dim","The journal has no file names, so it cannot be compared with git status."));
  else{
    if(!R.changed_outside_recorded_calls.length&&!R.deleted_but_present.length)rr.appendChild(el("div","ok","The recorded activity matches git status."));
    if(R.changed_outside_recorded_calls.length){rr.appendChild(el("div","warn","Changed outside the agent's recorded tool calls:"));var u=el("ul","plain mono");R.changed_outside_recorded_calls.forEach(function(p){u.appendChild(el("li","",p));});rr.appendChild(u);}
    if(R.deleted_but_present.length){rr.appendChild(el("div","warn","Recorded as deleted, but still present:"));var u2=el("ul","plain mono");R.deleted_but_present.forEach(function(p){u2.appendChild(el("li","",p));});rr.appendChild(u2);}
  }
  (D.partial||[]).forEach(function(n){rr.appendChild(el("div","dim","Note: "+n));});
}
function init(){
  $("title").textContent="Replay · "+D.agent+(D.meta.project?" · "+D.meta.project+(D.meta.branch?":"+D.meta.branch:""):"");
  $("sub").textContent=new Date(start).toLocaleString()+" · "+dur(Math.round((end-start)/1000));
  $("t0").textContent=hm(start);$("t1").textContent=hm(end);
  var ag=$("agent");[["",  "All agents"],[D.agent,D.agent]].forEach(function(o){var op=el("option","",o[1]);op.value=o[0];ag.appendChild(op);});
  buildMarkers();buildSummary();render();
  $("play").addEventListener("click",function(){setPlaying(!playing);});
  $("back").addEventListener("click",function(){setPlaying(false);pos=Math.max(0,pos-1);render();});
  $("fwd").addEventListener("click",function(){setPlaying(false);pos=Math.min(steps.length,pos+1);render();});
  $("speed").addEventListener("change",function(e){speed=parseFloat(e.target.value)||1;});
  $("pos").addEventListener("input",function(e){setPlaying(false);pos=parseInt(e.target.value,10)||0;render();});
  $("op").addEventListener("change",function(e){filt.op=e.target.value;render();});
  $("file").addEventListener("input",function(e){filt.file=e.target.value;render();});
  ag.addEventListener("change",function(e){filt.agent=e.target.value;render();});
  document.addEventListener("keydown",function(e){
    if(e.target&&(e.target.tagName==="INPUT"&&e.target.type==="text"||e.target.tagName==="SELECT"))return;
    if(e.key===" "&&e.target===document.body){e.preventDefault();setPlaying(!playing);}
    else if(e.key==="ArrowRight"&&e.target.type!=="range"){pos=Math.min(steps.length,pos+1);render();}
    else if(e.key==="ArrowLeft"&&e.target.type!=="range"){pos=Math.max(0,pos-1);render();}
  });
}
init();
})();
`
