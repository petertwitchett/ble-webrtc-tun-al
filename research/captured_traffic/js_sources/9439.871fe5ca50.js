try{!function(){var e="u">typeof window?window:"u">typeof global?global:"u">typeof globalThis?globalThis:"u">typeof self?self:{},t=(new e.Error).stack;t&&(e._sentryDebugIds=e._sentryDebugIds||{},e._sentryDebugIds[t]="47ed7290-70ac-4698-b5eb-6bb89cfe443a",e._sentryDebugIdIdentifier="sentry-dbid-47ed7290-70ac-4698-b5eb-6bb89cfe443a")}()}catch(e){}try{("u">typeof window?window:"u">typeof global?global:"u">typeof globalThis?globalThis:"u">typeof self?self:{}).SENTRY_RELEASE={id:"web@5.5.1+169491"}}catch(e){}"use strict";(self.rspackChunkweb=self.rspackChunkweb||[]).push([[9439],{63224(e,t,r){r.r(t),r.d(t,{default:()=>S});var i,n,o=r(701),l=r(42321),a=r(81680),c=r(4321),p=r(99290),s=r(41349),g=r(3115),m=r(86041),u=r(58132),d=r(95657),f=r(38440),h=r(97049),b=r(71747),x=r(10077),y=r(38344),$=r.n(y),v=r.p+"static/svg/reciept-bubble.b6a07c1d92.svg";(i=n||(n={})).Medium="Medium",i.Regular="Regular";let T={[n.Medium]:"500",[n.Regular]:"normal"},k=function(){let e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:n.Regular;return`
  font-family: IRANSans;
  font-weight: ${T[e]}
`},P=$().p`
  ${e=>`
     margin: 0;
    ${k(e.fontWeight??n.Regular)};
     font-size: ${e.fontSize}px;
     line-height: ${e.lineHeight}px;
     ${e.color?`color: ${e.color};`:""}
     ${e.margin?`margin: ${e.margin};`:""}
     ${e.marginRight?`margin-right: ${e.marginRight}px;`:""}
     ${e.marginLeft?`margin-left: ${e.marginLeft}px;`:""}
     ${e.marginBottom?`margin-bottom: ${e.marginBottom}px;`:""}
     ${e.marginTop?`margin-top: ${e.marginTop}px;`:""}
  `}
`;$().span`
  ${e=>`
     margin: 0;
    ${k(e.fontWeight??n.Regular)};
     font-size: ${e.fontSize}px;
     line-height: ${e.lineHeight}px;
     ${e.color?`color: ${e.color};`:""}
     ${e.margin?`margin: ${e.margin};`:""}
     ${e.marginRight?`margin-right: ${e.marginRight}px;`:""}
     ${e.marginLeft?`margin-left: ${e.marginLeft}px;`:""}
     ${e.marginBottom?`margin-bottom: ${e.marginBottom}px;`:""}
     ${e.marginTop?`margin-top: ${e.marginTop}px;`:""}
  `}
`,(0,p.css)`
  ${k(n.Regular)};
  font-size: 16px;
  line-height: 24px;
`,(0,p.css)`
  ${k(n.Regular)};
  font-size: 14px;
  line-height: 24px;
`;let w=$().div`
  background-color: ${e=>{let{theme:t}=e;return t.mobileChargeInternet.receipt.bg}};
  width: 328px;
  box-shadow: 0px 1px 2px rgba(9, 30, 66, 0.16);
  border-radius: 0 0 12px 12px;
  margin: 10px auto 16px auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  :before {
    content: "";
    background: url("${v}") no-repeat;
    background-size: 100%;
    position: absolute;
    top: 0;
    right: 0;
    left: 0;
    height: 7px;
    transform: translateY(-100%);
  }
`,Y=$().div`
  padding: 24px 16px 0 16px;
  align-self: stretch;
  display: flex;
  flex-direction: column;
  align-items: center;
`,z=$()(P)`
  color: ${e=>{let{theme:t,type:r}=e;return r&&"success"!==r?t["9"]:t.mobileChargeInternet.receipt.success_text}};
  margin-top: 20px;
  font-weight: 500;
  font-family: ${x.a};
`,R=$()(P)`
  color: ${e=>{let{theme:t}=e;return t["9"]}};
  margin: 0 0 24px 0;
  font-family: ${x.a};
`,I=$().div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  align-self: stretch;
  direction: rtl;
  margin-bottom: 16px;
  &:last-of-type {
    margin-bottom: 24px;
  }
`,_=$()(P)`
  color: ${e=>{let{theme:t}=e;return t.mobileChargeInternet.receipt.key_text}};
  font-family: ${x.a};
`,C=$()(P)`
  direction: ${e=>{let{dir:t}=e;return t||"rtl"}};
  color: ${e=>{let{theme:t}=e;return t.mobileChargeInternet.receipt.value_text}};
  font-family: ${x.a};
`;$().div`
  border-top: 1px solid
    ${e=>{let{theme:t}=e;return t.mobileChargeInternet.receipt.borderTop}};
  display: flex;
  align-items: center;
  align-self: stretch;
  direction: rtl;
`,$().div`
  flex-grow: 1;
  display: flex;
  direction: rtl;
  align-self: stretch;
  justify-content: center;
  padding: 14px 0;
  box-sizing: border-box;
  align-items: center;
  gap: 12px;
`,$()(P)`
  color: ${e=>{let{theme:t}=e;return t.mobileChargeInternet.receipt.shareText}};
  font-family: ${x.a};
`;let L=e=>{let{code:t,items:r,onPrimaryClick:i,onSecondaryClick:n,primaryText:l,transactionKey:a,onTransactionClicked:c,message:s,type:g,secondaryText:m,primaryIcon:u,title:d}=e,f=(0,p.useTheme)(),x=(0,o.Y)(h.M,{color:f.mobileChargeInternet.receipt.success_text,size:40}),y=(0,o.Y)(b.U,{size:40});return(0,o.Y)(w,{children:(0,o.FD)(Y,{children:[g&&"success"!=g?y:x,(0,o.Y)(z,{type:g,marginTop:0,marginBottom:24,fontSize:18,lineHeight:26,children:d}),s&&(0,o.Y)(R,{fontSize:15,lineHeight:24,children:s}),"error"!==g&&(0,o.Y)(o.FK,{children:r&&r.map(e=>(0,o.FD)(I,{children:[(0,o.Y)(_,{marginBottom:0,marginTop:0,fontSize:15,lineHeight:24,children:e.key}),(0,o.Y)(C,{dir:e.dir,marginTop:0,marginBottom:0,fontSize:15,lineHeight:24,children:e.value})]}))})]})})};var S=()=>{var e;let{TopBar:t}=(0,d.bE)(),r=(0,s.useLocation)(),i=(0,g.L)(),{close:n}=(0,g.h)(),h=(0,p.useTheme)(),b=null==r?void 0:r.state,x=m.P.receipt.title,y=m.P.receipt.keys.cost,$=m.P.receipt.errorTitle,v=m.P.receipt.successTitle;(null==b?void 0:b.receiptType)===u.Lc.Transfer?(x=x+" "+m.P.receipt.transfer,y=y+" "+m.P.receipt.transfer,$=$+" "+m.P.receipt.transfer,v=m.P.receipt.transfer+" "+v):(null==b?void 0:b.receiptType)===u.Lc.Purchase&&(x=x+" "+m.P.receipt.purchase,y=y+" "+m.P.receipt.purchase,$=$+" "+m.P.receipt.purchase,v=m.P.receipt.purchase+" "+v);let T=(null==b?void 0:b.type)==="success"?[{key:y,value:`${(0,a.hZ)((0,a._h)(null==b||null==(e=b.amount)?void 0:e.toString()))} ریال`},{key:m.P.receipt.keys.date,value:(0,a._h)((0,c.Yp)(((null==b?void 0:b.date)??0)*1e3).replace("،"," - ").slice(0,-3)),dir:"ltr"},{key:m.P.receipt.keys.destination,value:(0,a._h)((null==b?void 0:b.dstToken)??"")},{key:m.P.receipt.keys.origin,value:(0,a._h)(null==b?void 0:b.srcToken)}]:[];return(0,o.FD)(f.YW,{children:[(0,o.Y)(t,{title:x,onClose:()=>i("")}),(0,o.Y)(f.LN,{children:(0,o.Y)(L,{onPrimaryClick:n,primaryText:m.P.receipt.return,items:T,primaryIcon:(0,o.Y)(l.Q,{size:16,color:h.secondary_2}),type:null==b?void 0:b.type,message:(null==b?void 0:b.type)==="error"?m.P.receipt.errorDescription:"",title:(null==b?void 0:b.type)==="error"?$:v})})]})}}}]);
//# sourceMappingURL=9439.871fe5ca50.js.map