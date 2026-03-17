<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#readConfigurationTemp .container{
		padding-top: 30px;
		margin: 0px 80px;
	}
	#readConfigurationTemp .tempTitleCls{
		font-size: 14px;
		margin-bottom: 10px;
	}
	#readConfigurationTemp .tempTableCls{
		margin-bottom: 30px;
	}
	#readConfigurationTemp .readTempFootButton{
		height: 60px;
		line-height: 60px;
		padding-left: 80px;
		position: absolute;
		bottom: 0px;
		left: 0px;
		right: 0px;
		background-color: #FFFFFF;
		border: 1px solid #E9E9E9;
	}
	#readConfigurationTemp .el-ctable .no-data::before{
		display: none!important;
	}
</style>
<div class="pageDefault" id='readConfigurationTemp'>
	<div class="container">
        <div class="tempTitleCls">TFT List</div>
        <div class="tempTableCls">  <!-- :url="TFT_tableListUrl" :data="TFT_tableData" -->
            <el-ctable id="TFT_tableList" :url="TFT_tableListUrl" :height="height" pagination="true" :rownumber="true" :query-params="TFT_Params" style="border:1px solid #E9E9E9;">
				<el-table-column label='PF ID' prop="pfId"></el-table-column>
				<el-table-column label='APP Name' prop="appName"></el-table-column>
                <el-table-column label='Protocol' prop="protocal">
					<template slot-scope="scope">
						<div v-if="scope.row.protocal == '4'">IP</div>
						<div v-if="scope.row.protocal == '6'">TCP</div>
						<div v-if="scope.row.protocal == '17'">UDP</div>
					</template>
				</el-table-column>
				<el-table-column label='IP_MASK' prop="ipMask"></el-table-column>
                <el-table-column label='Port' prop="port"></el-table-column>
			</el-ctable>
        </div>
        <div class="tempTitleCls">QoS List</div>
        <div>  <!-- :url="QoS_tableListUrl" :data="QoS_tableData" -->
            <el-ctable id="QoS_tableList" ref="QoS_tableList" :url="QoS_tableListUrl" :height="height"  pagination="true" :rownumber="true" :query-params="QoS_Params" style="border:1px solid #E9E9E9;">
				<el-table-column label='PCC Name' prop="pccName"></el-table-column>
				<el-table-column label='QCI' prop="qci"></el-table-column>
                <el-table-column label='MBR_UL(bit/s)' prop="mbrUl"></el-table-column>
				<el-table-column label='MBR_DL(bit/s)' prop="mbrDl"></el-table-column>
                <el-table-column label='GRB_UL(bit/s)' prop="grbUl"></el-table-column>
                <el-table-column label='GRB_DL(bit/s)' prop="grbDl"></el-table-column>
				<el-table-column label='ARP_PL' prop="arpPl"></el-table-column>
                <el-table-column label='ARP_PCI' prop="arpPci">
					<template slot-scope="scope">
						<div v-if="scope.row.arpPci == '1'">On</div>
						<div v-if="scope.row.arpPci == '0'">Off</div>
					</template>
				</el-table-column>
				<el-table-column label='ARP_PVI' prop="arpPvi">
					<template slot-scope="scope">
						<div v-if="scope.row.arpPvi == '1'">On</div>
						<div v-if="scope.row.arpPvi == '0'">Off</div>
					</template>
				</el-table-column>
                <el-table-column label='Precedence' prop="precedence"></el-table-column>
                <el-table-column label='PF_List' prop="pfList"></el-table-column>
			</el-ctable>
        </div>
		<div class="readTempFootButton">
			<el-button type="primary" :disabled="QoS_tableAllDataTag" @click="applyTemplate"><%=rb.getString("YingYongMuBan")%></el-button>
			<el-button type="primary" :disabled="QoS_tableAllDataTag" @click="emptyTemplate"><%=rb.getString("QingKong")%></el-button>
			<el-button @click="closeTemplate"><%=rb.getString("QuXiao")%></el-button>
		</div>
    </div>
</div>
<script type="text/javascript">
new Vue({
	el:'#readConfigurationTemp',
	data(){
		var vm = this;
		
		return {
            TFT_tableListUrl:'${ctx}/cell/quicksettings/queryTFTTemplatePage.action',
			QoS_tableListUrl:'${ctx}/cell/quicksettings/queryQoSTemplatePage.action',
            height:'300px',
            TFT_Params:{
                timeZone:timeZone,
            },
            QoS_Params:{
                timeZone:timeZone,
            },
            TFT_tableData:[
                {PF_ID:'1',APP_NAME:'TEST1',PROTOCOL:'ip',IP_MASK:'192.168.0.1/24',PORT:''},
				{PF_ID:'2',APP_NAME:'TEST2',PROTOCOL:'tcp',IP_MASK:'192.168.0.1/18',PORT:'5060'},
				{PF_ID:'3',APP_NAME:'TEST3',PROTOCOL:'udp',IP_MASK:'192.168.0.1/10',PORT:'8081'},
				{PF_ID:'4',APP_NAME:'TEST4',PROTOCOL:'tcp',IP_MASK:'192.168.0.1/4',PORT:'6080'},
            ],
            QoS_tableData:[
                {PCC_NAME:'TEST1',QCI:'1:Example Services:Conversational voicemscbsc',MBR_UL:'104857600',MBR_DL:'104857601',GRB_UL:'104857600',GRB_DL:'104857601',ARP_PL:'15',ARP_PCI:'0',ARP_PVI:'0',PRECEDENCE:'255',PF_LIST:'1,2'},
				{PCC_NAME:'TEST2',QCI:'Conversational Video(Live streaming)',MBR_UL:'104857600',MBR_DL:'104857601',GRB_UL:'104857600',GRB_DL:'104857601',ARP_PL:'15',ARP_PCI:'1',ARP_PVI:'1',PRECEDENCE:'255',PF_LIST:''},
                {PCC_NAME:'TEST3',QCI:'Real Time Gaming',MBR_UL:'104857600',MBR_DL:'104857601',GRB_UL:'104857600',GRB_DL:'104857601',ARP_PL:'15',ARP_PCI:'0',ARP_PVI:'0',PRECEDENCE:'255',PF_LIST:'1,2'},
                {PCC_NAME:'TEST4',QCI:'Conversational Video(Live streaming)',MBR_UL:'104857600',MBR_DL:'104857601',GRB_UL:'104857600',GRB_DL:'104857601',ARP_PL:'15',ARP_PCI:'1',ARP_PVI:'0',PRECEDENCE:'255',PF_LIST:'1,2'},
                {PCC_NAME:'TEST5',QCI:'Real Time Gaming',MBR_UL:'104857600',MBR_DL:'104857601',GRB_UL:'104857600',GRB_DL:'104857601',ARP_PL:'15',ARP_PCI:'1',ARP_PVI:'0',PRECEDENCE:'255',PF_LIST:'1,2'},
            ],
			TFT_tableAllData:[],
			QoS_tableAllData:[],
			TFT_tableAllDataTag:false,
			QoS_tableAllDataTag:true,
			TFTList:[],
			QoSList:[],
		}
	},
	methods:{
		// 初始化请求数据
		init(){
			var vm = this;
			vm.getTFTTableData();
			
		},
		// 应用此模板 
		applyTemplate(){
			var vm = this;
			var tftTb = $('#17EF1A3DD71826B1E0DCED9FDDA82005'),
				qosTb = $('#C5F614F36FAFBAA281777B754312924D'),
				tftOpts = $(tftTb).datagrid('options'),
				qosOpts = $(qosTb).datagrid('options'),
				tftIdField = tftOpts.idField,
				qosIdField = qosOpts.idField,
				TFTList,QoSList;
			QoSList = $(qosTb).datagrid('getRows');
			TFTList = $(tftTb).datagrid('getRows');
			TFTList.map((item)=>{
				vm.TFTList.push(item);
			});
			QoSList.map((item)=>{
				vm.QoSList.push(item);
			})
			if(vm.TFTList.length !== 0){
				vm.TFTList.map((item,index)=>{
					var itemIndex =  $(tftTb).datagrid('getRowIndex',item);
					if(item.operateType == 'add'){

						Render.tableCollector['17EF1A3DD71826B1E0DCED9FDDA82005'].forEach((items,index,array)=>{
							if(items['BC3DC315DF2D8654F8995B0D8696C800'] == item['BC3DC315DF2D8654F8995B0D8696C800']){
								array.splice(items,1)
							}
						})
					}else{
						// if(Render.tableCollector['17EF1A3DD71826B1E0DCED9FDDA82005']){
						// 	Render.tableCollector['17EF1A3DD71826B1E0DCED9FDDA82005'].forEach((items,index,array)=>{
						// 		if(items['BC3DC315DF2D8654F8995B0D8696C800'] == item['BC3DC315DF2D8654F8995B0D8696C800']){
						// 			items.operateType = 'remove'
						// 		}
						// 	})
						// }else{
						// 	item.operateType = 'remove';
						// 	delete item._edit;
						// 	delete item._remove;
						// 	toTbDataQueue(tftTb, item, tftIdField);
						// }
						var params={
							operateType : 'remove',
							"7A8D21C159139F517BB9B592CF8E9ACA":item["7A8D21C159139F517BB9B592CF8E9ACA"]
						};
						toTbDataQueue(tftTb, params, tftIdField);
						
					}
					$(tftTb).datagrid('deleteRow',itemIndex);
				})
			}
			if(vm.QoSList.length !== 0){
				vm.QoSList.map((item,index)=>{
					var itemIndex =  $(qosTb).datagrid('getRowIndex',item);
					if(item.operateType == 'add'){

						Render.tableCollector['C5F614F36FAFBAA281777B754312924D'].forEach((items,index,array)=>{
							if(items['56C85E6D9E1F0229854130A5FAA47FFC'] == item['56C85E6D9E1F0229854130A5FAA47FFC']){
								array.splice(items,1)
							}
						})
					}else{
						// if(Render.tableCollector['C5F614F36FAFBAA281777B754312924D']){
						// 	Render.tableCollector['C5F614F36FAFBAA281777B754312924D'].forEach((items,index,array)=>{
						// 		if(items['56C85E6D9E1F0229854130A5FAA47FFC'] == item['56C85E6D9E1F0229854130A5FAA47FFC']){
						// 			items.operateType = 'remove'
						// 		}
						// 	})
						// }else{
						// 	item.operateType = 'remove';
						// 	delete item._edit;
						// 	delete item._remove;
						// 	toTbDataQueue(qosTb, item, qosIdField);
						// }
						var params={
							operateType : 'remove',
							"5540762DDECD826BBC6AED827744F0A2":item["5540762DDECD826BBC6AED827744F0A2"]
						};
						toTbDataQueue(qosTb, params, qosIdField);
						
					}
					$(qosTb).datagrid('deleteRow',itemIndex);
				})
			}
			vm.TFT_tableAllData.map((item,index)=>{
				item.operateType = 'add';
				$(tftTb).datagrid('appendRow', item);
				delete item._edit;
				delete item._remove;
				toTbDataQueue(tftTb, item, tftIdField);
			});
			vm.QoS_tableAllData.map((item,index)=>{
				item.operateType = 'add';
				$(qosTb).datagrid('appendRow', item);
				delete item._edit;
				delete item._remove;
				toTbDataQueue(qosTb, item, qosIdField);
			});
			openPropsPanel(false);
		},
		// 清空模板
		emptyTemplate(){
			var vm = this;
			vm.$confirm('<%=rb.getString("QueRenQingKongMuBan")%>','<%=rb.getString("QueRen")%>').then(function(){
				axios.post('${ctx}/cell/quicksettings/clearTftQosTemplate.action').then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							openPropsPanel(false);
						}else{
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
			});
			
		},
		// 请求TFT List表格数据
		getTFTTableData(){
			var vm = this,
				params={
					timeZone:timeZone,
				};
			axios.post('${ctx}/cell/quicksettings/queryTFTTemplate.action',stringify(params)).then(function(response){
				let data = response.data;
				var tftList = [];
				data.map((item)=>{
					var itemData={};
					itemData['BC3DC315DF2D8654F8995B0D8696C800'] = item.pfId||'';
					itemData['7A8D21C159139F517BB9B592CF8E9ACA'] = item.index;
					itemData['A0A25A7E46F515D47C97BA3D733F32B3'] = item.appName||'';
					itemData["CB674AAEFC770A68A7D3D5FEC007238E"] = item.protocal||'';
					itemData["827AE949AC32A97C1FA6F73AEA35A6C8"] = item.ipMask||'';
					itemData["F1A1CB61FAA12C13DF9E1E24B2B4EAF0"] = item.port||'';
					itemData._edit = true;
					itemData._remove = true;
					tftList.push(itemData);
				})
				vm.TFT_tableAllData = tftList;
				if(vm.TFT_tableAllData.length == 0){
					vm.TFT_tableAllDataTag = true;
				}
				vm.getQoSTableData();
				
			}).catch(function(error){})
		},
		// 请求QoS List表格数据
		getQoSTableData(){
			var vm = this,
				params={
					timeZone:timeZone,
				};

			axios.post('${ctx}/cell/quicksettings/queryQoSTemplate.action',stringify(params)).then(function(response){
				let data = response.data;
				var qosList = [];
				data.map((item)=>{
					var itemData={};
					itemData["D303A3555BED53E8FC5F34E883104904"] = item.pccId;
					itemData["56C85E6D9E1F0229854130A5FAA47FFC"] = item.pccName;
					itemData['5540762DDECD826BBC6AED827744F0A2'] = item.index;
					itemData["BE5E24E4B68D8037D68B7047C13C0A93"] = item.qci;
					itemData["702A697631DA87471CFBF0F6AE1B2DDA"] = item.mbrUl;
					itemData["A175EC59C27D988CED837D8A31D1AFD6"] = item.mbrDl;
					itemData["D4BD7B7FBCE7FE09C76FF9B2B0B63DF9"] = item.grbUl;
					itemData["0EAB6B40D32DBF65DE0A93345B517E01"] = item.grbDl;
					itemData["BB88A1C448BE32514BD74C8F6296B8F1"] = item.arpPl;
					itemData["FA7036C847AD76E239E9A0538CA0EE37"] = item.arpPci;
					itemData["D4C8710204B6D3D3EA5DF0D34C36B3E6"] = item.arpPvi;
					itemData["B4658365E4D73CD935B128E64F14F6AB"] = item.precedence;
					itemData["5A5BA201950EDCAB6F15E29DA226185A"] = item.pfList;
					itemData._edit = true;
					itemData._remove = true;
					qosList.push(itemData);
				})
				vm.QoS_tableAllData = qosList;
				vm.QoS_tableAllDataTag = false;
				
			}).catch(function(error){})
		},
		closeTemplate(){
			openPropsPanel(false);
		}
	},
	mounted(){
        eventBus.$off('read-init').$on('read-init',this.init);
	}
	
})

</script> 
