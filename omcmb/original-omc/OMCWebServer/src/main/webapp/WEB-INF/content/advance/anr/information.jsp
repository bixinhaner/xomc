<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style>
#anrDetailFaultPage {
    height:100%;
    overflow:auto;
    margin-left:80px;
    padding-top:20px;
    box-sizing:border-box;
}
#anrDetailFaultPage label{
	color:#999999;
	margin-right:5px;
	text-align:left;
	display:inline-block;
	width:165px;
}
#anrDetailFaultPage span{
	color:#333333;
}
#anrDetailFaultPage >div{
	margin-bottom:20px;
	font-size:14px;
	display: flex;
}
#anrDetailFaultPage div div:first-child{
	width: 200px;
	color:#999999;
	text-align: left;
}

#anrDetailFaultPage .newEventPic:before{
	color:#4D84FF;
	font-size:18px;
}
#anrDetailFaultPage .confirmPic:before{
	color:#67D972;
	font-size:18px;
}
#anrDetailFaultPage .refusePic:before{
	font-size:18px;
}
#anrDetailFaultPage .timeoutPic:before{
	color:#F2B354;
	font-size:18px;
}
#anrDetailFaultPage .errorColor:before{
	font-size:18px;
}
#anrDetailFaultPage .successColor:before{
	color:#67D972;
	font-size:18px;
}
</style>

<div id="anrDetailFaultPage">
	<div>
		<div><%=rb.getString("FuWuXiaoQuID")%></div>
		<div>{{serveCellId}}</div>
	</div>
	<div>
		<div><%=rb.getString("FuWuXiaoQuPCI")%></div>
		<div>{{scPCI}}</div>
	</div>
	<div>
		<div><%=rb.getString("FuWuXiaoQuSSB")%></div>
		<div>{{scFreq}}</div>
	</div>
	<div>
		<div><%=rb.getString("LinQuXiaoQuID")%></div>
		<div>{{neighborCellID}}</div>
	</div>
	<div>
		<div><%=rb.getString("LTELinQuPCI")%></div>
		<div>{{ncPCI}}</div>
	</div>
	<div>
		<div><%=rb.getString("LTELinXiaoQuPinDian")%></div>
		<div>{{ncFreq}}</div>
	</div>
	<div>
		<div><%=rb.getString("LinQuGenZongQuYuMa")%></div>
		<div>{{tac}}</div>
	</div>

	<div>
		<div><%=rb.getString("TianJiaTuJingInfo")%></div>
		<div v-if="addOrigin == 0">
			<span><%=rb.getString("ANRShanChu")%></span>
		</div>
		<div v-if="addOrigin == 1">
			<span><%=rb.getString("ANRKongKouTianJia")%></span>
		</div>
		<div v-if="addOrigin == 2">
			<span><%=rb.getString("ShuangXiangLinQuTianJia")%></span>
		</div>
	</div>	
	<div>
		<div><%=rb.getString("TianJiaShiJianInfo")%></div>
		<div>{{addTime}}</div>
	</div>	
	<div>
		<div><%=rb.getString("ShangBaoGaiLinQuMRDeRSRP")%></div>
		<div>{{rptRSRP}}</div>
	</div>
	<div>
		<div><%=rb.getString("ShangBaoMRLeiXing")%></div>
		<div v-if="mrType == 1">
			<span><%=rb.getString("ZhouQiXing")%></span>
		</div>
		<div v-if="mrType == 2">
			<span><%=rb.getString("ShiJianXing")%></span>
		</div>
	</div>
	<div v-if="activeName == 'arNr'">
		<div><%=rb.getString("ShangBaoMRShiJianZhongLei")%></div>
		<div v-if="mrEventType == 1">
			<span><%=rb.getString("A1ShiJian")%></span>
		</div>
		<div v-if="mrEventType == 2">
			<span><%=rb.getString("A2ShiJian")%></span>
		</div>
		<div v-if="mrEventType == 3">
			<span><%=rb.getString("A3ShiJian")%></span>
		</div>
		<div v-if="mrEventType == 4">
			<span><%=rb.getString("A4ShiJian")%></span>
		</div>
		<div v-if="mrEventType == 5">
			<span><%=rb.getString("A5ShiJian")%></span>
		</div>
		<div v-if="mrEventType == 6">
			<span><%=rb.getString("A6ShiJian")%></span>
		</div>
	</div>	
	<div v-if="activeName == 'nrLte'">
		<div><%=rb.getString("ShangBaoMRShiJianZhongLei")%></div>
		<div v-if="mrEventType == 1">
			<span><%=rb.getString("B1ShiJian")%></span>
		</div>
		<div v-if="mrEventType == 2">
			<span><%=rb.getString("B2ShiJian")%></span>
		</div>		
	</div>
	<div>
		<div><%=rb.getString("ZhuangTai")%></div>
		<div v-if="status == 0">
			<span class="el-icon el-icon-status-newEvent newEventPic"></span>
			<span><%=rb.getString("XinShiJian")%></span>
		</div>
		<div v-if="status == 1">
			<span class="el-icon el-icon-status-confirm confirmPic"></span>
			<span style='margin-left:5px;'><%=rb.getString("YiQueRen")%></span>
		</div>
		<div v-if="status == 2">
			<span class="el-icon el-icon-status-refuse refusePic"></span>
			<span><%=rb.getString("YiJuJue")%></span>
		</div>
		<div v-if="status == 3">
			<span class="el-icon el-icon-status-timeOut timeoutPic"></span>
			<span style='margin-left:5px;'><%=rb.getString("YiChaoShi")%></span>
		</div>
	</div>
	<div>
		<div><%=rb.getString("JieGuo")%></div>

		<div v-if="result == '1'">
			<span class="el-icon el-icon-circle-success successColor"></span>
			<span style='margin-left:5px;'><%=rb.getString("ChengGong")%></span>
		</div>
		<div v-else>
			<span class="el-icon el-icon-circle-close errorColor"></span>
			<span style='margin-left:5px;'><%=rb.getString("ShiBai")%></span>
		</div>
	</div>
	</div>
</div>
<script type="text/javascript">
new Vue({
	el:"#anrDetailFaultPage",
	data(){
		return{
			serveCellId:'',
			scPCI:'',
			scFreq:'',
			neighborCellID:'',
			ncPCI:'',
			ncFreq:'',
			tac:'',
			addOrigin:'',
			addTime:'',
			
			//新增3个
			rptRSRP:'',
			mrType:'',
			mrEventType:'',
			anrTimeout:'',
			status:'',
			result:'',
			timeZone:timeZone,
			activeName:''
		}
	},
	methods:{
		anrInfoInit(id,activeName){
			var vm = this,id;
			vm.activeName = activeName;
			axios.post('${ctx}/anr/queryInformation.action',stringify({
				timeZone:timeZone,
				anrId : id,
			})).then(function(response){					
				var data = response.data;
				if(data){
					vm.serveCellId = data.serveCellId;
					vm.scPCI= data.scPCI;
					vm.scFreq= data.scFreq;
					vm.neighborCellID= data.neighborCellID;
					vm.ncPCI= data.ncPCI;
					vm.ncFreq= data.ncFreq;
					vm.tac= data.tac;
					vm.addOrigin= data.addOrigin;
					vm.addTime = data.addTime;					
					vm.rptRSRP = data.rptRSRP;
					vm.mrType = data.mrType;
					vm.mrEventType = data.mrEventType;					
					vm.anrTimeout = data.anrTimeout;
					vm.status= data.status;
					vm.result = data.result;
				}				
			}) 						
		}
	},
	mounted(){
		eventBus.$off('view-config').$on('view-config',this.anrInfoInit);
	}	
})
</script>