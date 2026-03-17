<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>

<style>
#alarmDetailPage{
	height:100%;
	width: 100%;
	box-sizing:border-box;
}
#alarmDetailPage label{
	color:#999999;
	margin-right:5px;
	text-align:left;
	display:inline-block;
	width:200px;
}
#alarmDetailPage span{
	color:#333333;
	width: calc(100% - 240px);
	word-wrap: break-word;
}
#alarmDetailPage .alarmDetailMainBoxCls div{
	margin-bottom:20px;
	font-size:14px;
	display: flex;
	width: 100%;
}
.alarmDetailMainBoxCls{
	height:100%;
	width: 100%;
	padding: 20px 40px;
	box-sizing: border-box;
	overflow:auto;
}
</style>

<div id="alarmDetailPage">
	<div class="alarmDetailMainBoxCls">
		<div>
			<label><%=rb.getString("XuHao")%></label>
			<span>{{ALARM_ID}}</span>
		</div>
		<div>
			<label><%=rb.getString("GaoJingWeiYiBiaoZhi")%></label>
			<span>{{ALARM_IDENTIFIER}}</span>
		</div>
		<div>
			<label><%=rb.getString("KeNengYuanYin")%></label>
			<span>{{ALARM_NAME}}</span>
		</div>
		<div>
			<label><%=rb.getString("JuTiGuZhang")%></label>
			<span>{{SPECIFIC_PROBLEM}}</span>
		</div>
		<div>
			<label><%=rb.getString("FuJianXinXi")%></label>
			<span>{{ADDITIONAL_INFORMATION}}</span>
		</div>
		<div>
			<label><%=rb.getString("FuJianWenBen")%></label>
			<span>{{ADDITIONAL_TEXT}}</span>
		</div>
		<div>
			<label><%=rb.getString("YanZhongChengDu")%></label>
			<span>{{ALARM_SERVERITY}}</span>
		</div>
		<div>
			<label><%=rb.getString("ShiJianLeiXing")%></label>
			<span>{{EVENT_TYPE}}</span>
		</div>
		<div>
			<label><%=rb.getString("XinGaoJingYuan")%></label>
			<span>{{NE_TYPE}}</span>
		</div>
		<div>
			<label><%=rb.getString("WangYuanDingWei")%></label>
			<span>{{EQUIP_INFO}}</span>
		</div>
		<div>
			<label><%=rb.getString("GaoJingZhuangTai")%></label>
			<span>{{DEAL_STATE}}</span>
		</div>
		<div>
			<label><%=rb.getString("GuZhangShiJian")%></label>
			<span>{{EVENT_TIME}}</span>
		</div>
		<div>
			<label><%=rb.getString("GengXinShiJian")%></label>
			<span>{{UPD_TIME}}</span>
		</div>
		<div v-if="confirmFlag">
			<label><%=rb.getString("QueRenRen")%></label>
			<span>{{DEAL_USER}}</span>
		</div>
		<div v-if="confirmFlag">
			<label><%=rb.getString("QueRenShiJian")%></label>
			<span>{{DEAL_TIME}}</span>
		</div>
		<div v-if="clearFlag">
			<label><%=rb.getString("GaoJingQingChuRen")%></label>
			<span>{{CLEAR_USER}}</span>
		</div>
		<div v-if="clearFlag">
			<label><%=rb.getString("GaoJingQingChuShiJian")%></label>
			<span>{{ClEAR_TIME}}</span>
		</div>
		<div style="display: flex;">
			<label style="min-width: 165px"><%=rb.getString("ChuLiJianYi")%></label>
			<span>{{SUGGESTION}}</span>
		</div>
		<div>
			<label><%=rb.getString("MiaoShu")%></label>
			<span>{{DEAL_MEMO}}</span>
		</div>
	</div>
</div>
<script type="text/javascript">

var alarmDetailVue =  new Vue({
	el:"#alarmDetailPage",
	data(){
		return{
			"ALARM_ID":'',
			"ALARM_IDENTIFIER":'',
			"ALARM_NAME":'',
			"ADDITIONAL_INFORMATION":'',
			"ADDITIONAL_TEXT":'',
			"SPECIFIC_PROBLEM":'',
			"ALARM_SERVERITY":'',
			"EVENT_TYPE":'',
			"NE_TYPE":'',
			"EQUIP_INFO":'',
			"DEAL_STATE":'',
			"EVENT_TIME":'',
			"UPD_TIME":'',
			"DEAL_USER":'',
			"DEAL_TIME":'',
			"DEAL_MEMO":'',
			"CLEAR_USER":'',
			"ClEAR_TIME":'',
			"SUGGESTION":'',
			confirmFlag:false,
			clearFlag:false,
			timeZone:timeZone
		}
	},
	methods:{
		init(faultType,alarm_id){
			var vm = this;
			 
				
			axios.post('${ctx}/cell/fault/queryAlarmDetail.action',stringify({
    			alarm_id : alarm_id,
    			type:faultType,
    			timeZone:vm.timeZone
	    	})).then(function(response){
	    		var data = response.data;
	    		
	    		
	    		 /* data = {
						"ALARM_ID":'38',
						"ALARM_IDENTIFIER":'7',
						"ALARM_NAME":'NE Disconnected',
						"SPECIFIC_PROBLEM":'Smallcell is disconnnectd',
						"ALARM_SERVERITY":'Critical',
						"EVENT_TYPE":'Communications Alarm',
						"NE_TYPE":'OMC',
						"EQUIP_INFO":'SN=234dfsd34;CellName=unknown name',
						"DEAL_STATE":'Unconfirmed and active',
						"EVENT_TIME":'2019-10-14 09:37:42',
						"UPD_TIME":'2019-10-15 02:242:11',
						"DEAL_USER":'',
						"DEAL_TIME":'',
						"CLEAR_USER":'',
						"ClEAR_TIME":'',
						"SUGGESTION":'1.Check if the GPS antenna is faulty.',
						"DEAL_INT_STATE":'0'
					} 
	    		 */
	    		
	    		vm.ALARM_ID = data.ALARM_ID;
				vm.ALARM_IDENTIFIER = data.ALARM_IDENTIFIER;
				vm.ALARM_NAME = data.ALARM_NAME;
				vm.ADDITIONAL_INFORMATION = data.ADDITIONAL_INFORMATION;
				vm.ADDITIONAL_TEXT = data.ADDITIONAL_TEXT;
				vm.SPECIFIC_PROBLEM = data.SPECIFIC_PROBLEM;
				vm.ALARM_SERVERITY = data.ALARM_SERVERITY;
				vm.EVENT_TYPE = data.EVENT_TYPE;
				vm.NE_TYPE = data.NE_TYPE;
				vm.EQUIP_INFO = data.EQUIP_INFO;
				vm.DEAL_STATE = data.DEAL_STATE;
				vm.EVENT_TIME = data.EVENT_TIME;
				vm.UPD_TIME = data.UPD_TIME;
				vm.DEAL_USER = data.DEAL_USER;
				vm.DEAL_TIME = data.DEAL_TIME;
				vm.CLEAR_USER = data.CLEAR_USER;
				vm.ClEAR_TIME = data.ClEAR_TIME;
				vm.SUGGESTION = data.SUGGESTION;
				vm.DEAL_MEMO = data.DEAL_MEMO;
				// 1,3 已确认 
				if( data.DEAL_INT_STATE == '1' || data.DEAL_INT_STATE == '3'){
					vm.confirmFlag = true
				}
				// 2,3 已清除
				if( data.DEAL_INT_STATE == '3' || data.DEAL_INT_STATE == '2'){
					vm.clearFlag = true
				}
	    	}) 
			
			
		}
	},
	mounted(){
		eventBus.$off('detail-info').$on('detail-info',this.init);
	
	}
	
})
</script>