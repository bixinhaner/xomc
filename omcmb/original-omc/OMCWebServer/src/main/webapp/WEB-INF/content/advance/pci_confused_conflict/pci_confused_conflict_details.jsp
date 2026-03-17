<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>

<style>

#pciConfusedConflictDetailsPage label{
	color:#999999;
	margin-right:5px;
	text-align:left;
	display:inline-block;
	width:165px;
}
#pciConfusedConflictDetailsPage span{
	color:#333333;
}
#pciConfusedConflictDetailsPage >div{
	margin-bottom:20px;
	font-size:14px;
	display: flex;
}
#pciConfusedConflictDetailsPage div div:first-child{
	width: 160px;
	color:#999999;
	text-align: left;
}
.PciConfusedSolve .el-icon:before{
		color:#67D972;
}
.PciConfusedNoSolve .el-icon:before{
	color:#E88282;
}
</style>

<div id="pciConfusedConflictDetailsPage" style="height:100%;overflow:auto;margin-left:80px;padding-top:20px;box-sizing:border-box;">
	<div>
		<div><%=rb.getString("XiaoZhanBianMa")%>:</div>
		<div>{{serialNumber}}</div>
	</div>
	<div>
		<div><%=rb.getString("HostName")%>:</div>
		<div>{{cellName}}</div>
	</div>
	<div>
		<div>Cell ECGI:</div>
		<div>{{cellEcgi}}</div>
	</div>
	<div>
		<div>Cell Earfcn:</div>
		<div>{{cellEarfcn}}</div>
	</div>
	<div>
		<div>Cell PCI:</div>
		<div>{{cellPci}}</div>
	</div>
	<div>
		<div><%=rb.getString("Type")%>:</div>
		<div class="PciConfusedSolve" v-if="type == 0">
			<span><%=rb.getString("PCIChongTu")%></span>
		</div>
		<div class="PciConfusedNoSolve" v-if="type == 1">
			<span><%=rb.getString("PCIHunXiao")%></span>
		</div>
	</div>
	<div>
		<div>Suggest PCI:</div>
		<div>{{suggestPci}}</div>
	</div>
	<div>
		<div>Neighbor Cell:</div>
		<div>{{neighborCell}}</div>
	</div>
	<div>
		<div><%=rb.getString("ZhuangTai")%>:</div>
		<div class="PciConfusedSolve" v-if="status == 1">
			<span class="el-icon el-icon-status-success" style='margin-right:5px;'></span><span><%=rb.getString("YiJieJue")%></span>
		</div>
		<div class="PciConfusedNoSolve" v-if="status == 0">
			<span class="el-icon el-icon-status-failed"></span><span style='margin-left:5px;'><%=rb.getString("WeiJieJue")%></span>
		</div>
	</div>
	<div>
		<div><%=rb.getString("GuZhangShiJian")%>:</div>
		<div>{{eventTime}}</div>
	</div>
	<div>
		<div><%=rb.getString("GengXinShiJian")%>:</div>
		<div>{{updateTime}}</div>
	</div>
	<div>
		<div><%=rb.getString("JieShuShiJian")%>:</div>
		<div>{{endTime}}</div>
	</div>
	<div>
		<div><%=rb.getString("TiaoShu")%>:</div>
		<div>{{count}}</div>
	</div>
	<div>
		<div><%=rb.getString("SheBeiZu")%>:</div>
		<div>{{deviceGroup}}</div>
	</div>
</div>
<script type="text/javascript">
new Vue({
	el:"#pciConfusedConflictDetailsPage",
	data(){
		return{
			serialNumber:'',
			cellName:'',
			cellEcgi:'',
			cellEarfcn:'',
			cellPci:'',
			type:'',
			suggestPci:'',
			neighborCell:'',
			status:'',
			eventTime:'',
			updateTime:'',
			endTime:'',
			count:'',
			deviceGroup:'',
			timeZone:timeZone,
			data:[
					{
						id:1,
						serialNumber:'12359642342422280',
						cellName:'zhang',
						cellEcgi:'32590',
						cellEarfcn:'80',
						cellPci:'55',
						type:'PCI冲突',
						suggestPci:'12',
						neighborCell:'Cell1 Name=name1,cell1-ECGI=254,CELL1-PCI=21',
						status:'1',
						eventTime:'2020-12-04 10:44:00',
						updateTime:'2020-12-04 11:55:00',
						count:'12',
						deviceGroup:'device Group'
					},
					{
						id:2,
						serialNumber:'12359642342422281',
						cellName:'xin',
						cellEcgi:'66655',
						cellEarfcn:'10',
						cellPci:'33',
						type:'PCI冲突',
						suggestPci:'12',
						neighborCell:'Cell1 Name=name1,cell1-ECGI=254,CELL1-PCI=21',
						status:'2',
						eventTime:'2020-12-04 10:44:00',
						updateTime:'2020-12-04 11:55:00',
						count:'3',
						deviceGroup:'device Group'
					},
					{
						id:3,
						serialNumber:'12359642342422282',
						cellName:'a',
						cellEcgi:'66655',
						cellEarfcn:'10',
						cellPci:'33',
						type:'PCI冲突',
						suggestPci:'12',
						neighborCell:'Cell1 Name=name1,cell1-ECGI=254,CELL1-PCI=21',
						status:'2',
						eventTime:'2020-12-04 10:44:00',
						updateTime:'2020-12-04 11:55:00',
						count:'4',
						deviceGroup:'device Group'
					}
				]
		}
	},
	methods:{
		init(id){
			var vm = this,id;
			// id = id-1;
			// vm.serialNumber = vm.data[id].serialNumber;
			// vm.cellName= vm.data[id].cellName;
			// vm.cellEcgi= vm.data[id].cellEcgi;
			// vm.cellEarfcn= vm.data[id].cellEarfcn;
			// vm.cellPci= vm.data[id].cellPci;
			// vm.type= vm.data[id].type;
			// vm.suggestPci= vm.data[id].suggestPci;
			// vm.neighborCell= vm.data[id].neighborCell;
			// vm.status= vm.data[id].status;
			// vm.eventTime= vm.data[id].eventTime;
			// vm.updateTime= vm.data[id].updateTime;
			// vm.count= vm.data[id].count;
			// vm.deviceGroup= vm.data[id].deviceGroup;
			axios.post('${ctx}/pci/queryInformation.action',stringify({
					timeZone:timeZone,
					id : id,
				})).then(function(response){
					var data = response.data;
					vm.serialNumber = data.smallCellCode;
					vm.cellName= data.cellName;
					vm.cellEcgi= data.cellEcgi;
					vm.cellEarfcn= data.cellEarfcn;
					vm.cellPci= data.cellPci;
					vm.type= data.type;
					vm.suggestPci= data.suggestPci;
					vm.neighborCell= data.neighborCell;
					vm.status= data.status;
					vm.eventTime= data.eventTime;
					vm.updateTime= data.updateTime;
					vm.endTime = data.endTime;
					vm.count= data.count;
					vm.deviceGroup= data.deviceGroup;
			}) 
			
			
		}
	},
	mounted(){
		eventBus.$off('detail-info').$on('detail-info',this.init);
	
	}
	
})
</script>