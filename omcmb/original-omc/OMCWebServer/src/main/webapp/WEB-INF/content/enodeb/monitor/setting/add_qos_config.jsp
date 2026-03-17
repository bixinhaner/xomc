<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#QosAddPage .container{
		padding-top: 50px;
	}
	#QosAddPage .addQosFormItem{
		display: flex;
		margin-left:80px;
	}
	#QosAddPage .addQosFormItem .el-form-item{
		width: 600px;
		margin-bottom: 20px;
	}
	#QosAddPage .addQosFormItem .el-form-item__error{
		padding-top: 0px;
	}
	#QosAddPage .addQosFormTable .el-form-item{
		margin-bottom: 0px;
	}
	#QosAddPage .addQosFootButton{
		height: 60px;
		line-height: 60px;
		padding-left: 80px;
		position: absolute;
		bottom: 0px;
		left: 0px;
		right: 0px;
        background-color: #FFFFFF;
		border: 1px solid #E9E9E9;
		z-index: 99;
	}
	#QosAddPage .QciClass .el-input{
		width: 320px!important;
	}
	#QosAddPage .promptInfoParentStys div:first-child{
		overflow: hidden;
	}
	#QosAddPage .promptInfo{
		font-size: 12px;
		color: #BBBBBB;
		position: relative;
		display: inline-block;
		margin-left: 5px;
		top:-8px;
	}
	#QosAddPage .promptInfo .el-icon:before{
		color: #BBBBBB;
		font-size: 12px;
	}
	#QosAddPage .el-pairgrid-title{
		top: -20px!important;
	}
	#QosAddPage .el-pairgrid .no-data::before{
		display: none!important;
	}
</style>
<div class="pageDefault" id='QosAddPage'>
	<div class="container">
        <el-form :model="addQosForm" ref="addQosForm" :rules="rules"  label-position="top" label-width="120px" :hide-required-asterisk='true'>
			<div class="addQosFormItem">
				<el-form-item label="PCC ID" prop="PCC_ID">
					<el-input v-model="addQosForm.PCC_ID" style="padding-top:5px;" placeholder="<%=rb.getString("FanWei")%>:1-100"></el-input>
				</el-form-item>
				<el-form-item label="PCC Name" prop="PCC_NAME">
					<el-input v-model="addQosForm.PCC_NAME" style="padding-top:5px;" placeholder="<%=rb.getString("ZiFuChang")%><%=rb.getString("MaoHao")%>1-40"></el-input>
				</el-form-item>
			</div>
			<div class="addQosFormItem">
				<el-form-item  label="MBR UL(bit/s)" prop="MBR_UL">
					<el-input  v-model="addQosForm.MBR_UL" style="padding-top:5px;" placeholder="<%=rb.getString("FanWei")%>:1000-104857600"></el-input>
				</el-form-item>
				<el-form-item  label="MBR DL(bit/s)" prop="MBR_DL">
					<el-input  v-model="addQosForm.MBR_DL" style="padding-top:5px;" placeholder="<%=rb.getString("FanWei")%>:1000-104857600"></el-input>
				</el-form-item>
			</div>
			<div class="addQosFormItem">
				<el-form-item  label="GRB UL(bit/s)" prop="GRB_UL">
					<el-input  v-model="addQosForm.GRB_UL"  style="padding-top:5px;" placeholder="<%=rb.getString("FanWei")%>:1000-104857600"></el-input>
				</el-form-item>
				<el-form-item  label="GRB DL(bit/s)" prop="GRB_DL">
					<el-input  v-model="addQosForm.GRB_DL"  style="padding-top:5px;" placeholder="<%=rb.getString("FanWei")%>:1000-104857600"></el-input>
				</el-form-item>
			</div>
			<div class="addQosFormItem">
				<el-form-item  label="ARP PL" prop="ARP_PL" class="promptInfoParentStys">
					<el-input  v-model="addQosForm.ARP_PL"  style="padding-top:5px;width:100px" placeholder="<%=rb.getString("FanWei")%>:1-15"></el-input>
					<el-tooltip placement="bottom">
						<div slot="content">
							<div><%=rb.getString("ZuiGaoYouXianJiTiShiOne")%></div>
							<div><%=rb.getString("ZuiGaoYouXianJiTiShiTwo")%></div>
						</div>
						<div class="promptInfo" ><span class="el-icon-circle-info el-icon"></span></div>
					</el-tooltip>
					
				</el-form-item>
				<el-form-item  label="ARP PCI" prop="ARP_PCI" class="promptInfoParentStys">
					<el-select v-model="addQosForm.ARP_PCI" style="padding-top:5px;">
						<el-option label="<%=rb.getString("Guan")%>" value="0"></el-option>
						<el-option label="<%=rb.getString("Kai")%>" value="1"></el-option>
					</el-select>
					<el-tooltip placement="bottom">
						<div slot="content">
							<div><%=rb.getString("Guan")%>:<%=rb.getString("KeBeiQiangZhanTiShi")%></div>
							<div><%=rb.getString("Kai")%>:<%=rb.getString("BuKeBeiQiangZhanTiShi")%></div>
						</div>
						<div class="promptInfo" ><span class="el-icon-circle-info el-icon"></span></div>
					</el-tooltip>
				</el-form-item>
			</div>
			<div class="addQosFormItem">
				<el-form-item  label="ARP PVI" prop="ARP_PVI" class="promptInfoParentStys">
					<el-select v-model="addQosForm.ARP_PVI" style="padding-top:5px;">
						<el-option label="<%=rb.getString("Guan")%>" value="0"></el-option>
						<el-option label="<%=rb.getString("Kai")%>" value="1"></el-option>
					</el-select>
					<el-tooltip placement="bottom">
						<div slot="content">
							<div><%=rb.getString("Guan")%>:<%=rb.getString("YunXuQiangZhanTiShi")%></div>
							<div><%=rb.getString("Kai")%>:<%=rb.getString("BuYunXuQiangZhanTiShi")%></div>
						</div>
						<div class="promptInfo" ><span class="el-icon-circle-info el-icon"></span></div>
					</el-tooltip>
				</el-form-item>
				<el-form-item  label="Precedence" prop="PRECEDENCE">
					<el-input  v-model="addQosForm.PRECEDENCE"  style="padding-top:5px;" placeholder="<%=rb.getString("FanWei")%>:1-254"></el-input>
				</el-form-item>
			</div>
			<div class="addQosFormItem">
				<el-form-item  label="QCI" prop="QCI" class="QciClass">
					<el-select v-model="addQosForm.QCI" style="padding-top:5px;" @change="QciChange">
						<el-option v-for="item in QciOptions" :key="item.value" :label="item.text" :value="item.value"></el-option>
					</el-select>
				</el-form-item>
			</div>
			<div class="addQosFormTable">
				<el-form-item  label="PF List" style="margin-left:80px; margin-bottom: 60px;">
					<el-pairgrid id="PF_List"  ref="cpairgrid"  @selection-change='selectChange' :limit="10" :grid-data="tftTableList"  :title="tableTitle" :height="height" row-key="BC3DC315DF2D8654F8995B0D8696C800" style="margin-right:80px;">
						<template slot="left">
							<el-table-column type="selection" width="45" :reserve-selection="true"></el-table-column>
							<el-table-column prop='BC3DC315DF2D8654F8995B0D8696C800' label='PF ID'></el-table-column>
							<el-table-column prop='A0A25A7E46F515D47C97BA3D733F32B3' label='APP Name'></el-table-column>
						</template>
						
						<template slot='right'>
							<el-table-column prop='BC3DC315DF2D8654F8995B0D8696C800' label='PF ID'></el-table-column>
							<el-table-column prop='A0A25A7E46F515D47C97BA3D733F32B3' label='APP Name'></el-table-column>
						</template>
					</el-pairgrid>
				</el-form-item>
			</div>
			
			<el-form-item prop='cellCodes' style="margin-left:80px;margin-bottom: 30px;">
				<el-input v-model='addQosForm.cellCodes' v-show="false"></el-input>
			</el-form-item>
		</el-form>
		<div class="addQosFootButton">
			<el-button @click='addQoSSubmit' type="primary"><%=rb.getString("QueDing")%></el-button>
			<el-button @click='closeAdd'><%=rb.getString("QuXiao")%></el-button>
		</div>
    </div>
</div>
<script type="text/javascript">
var tftTb = $('#17EF1A3DD71826B1E0DCED9FDDA82005'),
	qosTb = $('#C5F614F36FAFBAA281777B754312924D');
new Vue({
	el:'#QosAddPage',
	data(){
		var vm = this;
		var validate_PCC_ID = (rule,value,callback) => {

			if(value === ''){
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1-100'))
			}else if(vm.isNumeric(value)&& parseInt(value)>=1 && parseInt(value)<=100){
				if(value == vm.old_PCC_ID){
					callback();
				}else{
					if(vm.pccIdList.includes(value) == true){
						callback(new Error('<%=rb.getString("YiCunZai")%>'))
					}else{
						callback();
					}
				}
				
			}else{
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1-100'))
			}
		};
		var validate_PCC_NAME = (rule,value,callback) => {
			var reg = /^[\w+$]{1,40}$/;
			if(value === ''){
				callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXian")%><%=rb.getString("ZiFuChang")%><%=rb.getString("MaoHao")%>1-40'))
			}else if(reg.test(value)){
				if(vm.pccNameList.length == 0){
					callback();
				}else{
					if(vm.pccNameList.includes(value) == true){
						callback(new Error('<%=rb.getString("MingChengChongFu")%>'))
					}else{
						callback();
					}
				}
			}else{
				callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXian")%><%=rb.getString("ZiFuChang")%><%=rb.getString("MaoHao")%>1-40'))
			}
		};
		var validate_MBR_UL = (rule,value,callback) => {

			if(value === ''){
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1000-104857600'))
			}else if(vm.isNumeric(value)&& parseInt(value)>=1000 && parseInt(value)<=104857600){
				callback();
			}else{
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1000-104857600'))
			}
		};
		var validate_MBR_DL = (rule,value,callback) => {

			if(value === ''){
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1000-104857600'))
			}else if(vm.isNumeric(value)&& parseInt(value)>=1000 && parseInt(value)<=104857600){
				callback();
			}else{
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1000-104857600'))
			}
		};
		var validate_GRB_UL = (rule,value,callback) => {

			if(value === ''){
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1000-104857600'))
			}else if(vm.isNumeric(value)&& parseInt(value)>=1000 && parseInt(value)<=104857600){
				callback();
			}else{
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1000-104857600'))
			}
		};
		var validate_GRB_DL = (rule,value,callback) => {

			if(value === ''){
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1000-104857600'))
			}else if(vm.isNumeric(value)&& parseInt(value)>=1000 && parseInt(value)<=104857600){
				callback();
			}else{
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1000-104857600'))
			}
		};
		var validate_ARP_PL = (rule,value,callback) => {

			if(value === ''){
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1-15'))
			}else if(vm.isNumeric(value)&& parseInt(value)>=1 && parseInt(value)<=15){
				callback();
			}else{
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1-15'))
			}
		};
		var validate_ARP_PCI = (rule,value,callback) => {

			if(value === ''){
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>[0:1]'))
			}else if(vm.isNumeric(value)&& parseInt(value)>=0 && parseInt(value)<=1){
				callback();
			}else{
				callback(new Error('<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>[0:1]'))
			}
		};
		var validate_ARP_PVI = (rule,value,callback) => {

			if(value === ''){
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>[0:1]'))
			}else if(vm.isNumeric(value)&& parseInt(value)>=0 && parseInt(value)<=1){
				callback();
			}else{
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>[0:1]'))
			}
		};
		var validate_PRECEDENCE = (rule,value,callback) => {

			if(value === ''){
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>0-255'))
			}else if(vm.isNumeric(value)&& parseInt(value)>=1 && parseInt(value)<=254){
				
				if(vm.precedenceList.includes(value) == true){
					callback(new Error('<%=rb.getString("YiCunZai")%>'))
				}else{
					callback();
				}
			}else{
				callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>1-254'))
			}
		};
		return {
           	addQosForm:{
				PCC_ID:'',
                PCC_NAME:'',
                QCI:'1',
                MBR_UL:'',
                MBR_DL:'',
                GRB_UL:'',
                GRB_DL:'',
				ARP_PL:'1',
                ARP_PCI:'1',
				ARP_PVI:'0',
				PRECEDENCE:'',
				INDEX:'',
				PF_List:'',
				cellCodes:''
           	},
			portDisabledTag:true,
			QciOptions:[
				{value:'1',text:'1:Conversational voice'},
				{value:'2',text:'2:Conversational Video(Live streaming)'},
				{value:'3',text:'3:Real Time Gaming'},
				{value:'4',text:'4:Non-conversational voice(buffered streaming)'},
			],
			rules:{
				PCC_ID:[
					{validator:validate_PCC_ID,trigger:'blur'},
				],
				PCC_NAME:[
					{validator:validate_PCC_NAME,trigger:'blur'},
				],
				MBR_UL:[
					{validator:validate_MBR_UL,trigger:'blur'},
				],
				MBR_DL:[
					{validator:validate_MBR_DL,trigger:'blur'},
				],
				GRB_UL:[
					{validator:validate_GRB_UL,trigger:'blur'},
				],
				GRB_DL:[
					{validator:validate_GRB_DL,trigger:'blur'},
				],
				ARP_PL:[
					{validator:validate_ARP_PL,trigger:'blur'},
				],
				ARP_PCI:[
					{validator:validate_ARP_PCI,trigger:'blur'},
				],
				ARP_PVI:[
					{validator:validate_ARP_PVI,trigger:'blur'},
				],
				PRECEDENCE:[
					{validator:validate_PRECEDENCE,trigger:'blur'},
				],
				cellCodes:[
					{required:true,message:'<%=rb.getString("QingXuanZePF")%>',trigger:'change'}
				],
			},
			tftTableList:{},
			qosTableList:'',
			rightUrl:'',
			height:'300px',
			tableTitle:['','Selected'],
			TFTList:[],
			QoSList:[],
			precedenceList:[],
			pccNameList:[],
			pccIdList:[],
			rowData:'',
			operType:'',
			selectionData:'',
			selectedPfList:[],
			old_PCC_ID:''
		}
	},
	methods:{
		init(opts,row,type){
			var vm = this;
			vm.rowData = row;
			vm.operType = type;

			vm.getTFTAndQoSTableList();
			Render.validResult.reboot = false;
		},
		QciChange(){

		},
		selectChange(selection){
			var vm = this,ids=[];
			vm.selectionData = selection;
			vm.selectionData.map((item)=>{
				ids.push(item["BC3DC315DF2D8654F8995B0D8696C800"])
			});
			vm.addQosForm.PF_List = ids.join(',');

			vm.addQosForm.cellCodes = ids.join(',');
		},
		// 新增提交
		addQoSSubmit(){
			var vm = this;
			vm.$refs["addQosForm"].validate( valid => {
				if(valid){
					var parmas ={
							"D303A3555BED53E8FC5F34E883104904":vm.addQosForm.PCC_ID,
							"56C85E6D9E1F0229854130A5FAA47FFC":vm.addQosForm.PCC_NAME,
							"5540762DDECD826BBC6AED827744F0A2":vm.addQosForm.PRECEDENCE,
							"BE5E24E4B68D8037D68B7047C13C0A93":vm.addQosForm.QCI,
							"702A697631DA87471CFBF0F6AE1B2DDA":vm.addQosForm.MBR_UL,
							"A175EC59C27D988CED837D8A31D1AFD6":vm.addQosForm.MBR_DL,
							"D4BD7B7FBCE7FE09C76FF9B2B0B63DF9":vm.addQosForm.GRB_UL,
							"0EAB6B40D32DBF65DE0A93345B517E01":vm.addQosForm.GRB_DL,
							"BB88A1C448BE32514BD74C8F6296B8F1":vm.addQosForm.ARP_PL,
							"FA7036C847AD76E239E9A0538CA0EE37":vm.addQosForm.ARP_PCI,
							"D4C8710204B6D3D3EA5DF0D34C36B3E6":vm.addQosForm.ARP_PVI,
							"B4658365E4D73CD935B128E64F14F6AB":vm.addQosForm.PRECEDENCE,
							"5A5BA201950EDCAB6F15E29DA226185A":vm.addQosForm.PF_List,
						};
					if(vm.operType == 'add'){
						parmas.operateType = 'add';
						parmas["5540762DDECD826BBC6AED827744F0A2"] = vm.addQosForm.PRECEDENCE;
						vm.setValidResult();
					}else{
						parmas["5540762DDECD826BBC6AED827744F0A2"] = vm.rowData["5540762DDECD826BBC6AED827744F0A2"];
						if(vm.rowData.operateType == 'add'){
							parmas.operateType = 'add';
							Render.tableCollector['C5F614F36FAFBAA281777B754312924D'].forEach((items,index,array)=>{
								if(items['56C85E6D9E1F0229854130A5FAA47FFC'] == vm.rowData['56C85E6D9E1F0229854130A5FAA47FFC']){
									array.splice(items,1)
								}
							})
							$(qosTb).datagrid('deleteRow', vm.addQosForm.INDEX);
						}else{
							parmas.operateType = 'edit';
							Render.validResult.reboot = true;
						}
					}
					var opts = $(qosTb).datagrid('options'),idField = opts.idField;
					if(vm.operType == "add"){
						parmas._edit = true;
						parmas._remove = true;
						$(qosTb).datagrid('appendRow', parmas);
					}else{
						parmas._edit = true;
						parmas._remove = true;
						if(vm.rowData.operateType == 'add'){
							$(qosTb).datagrid('appendRow', parmas);
						}else{
							$(qosTb).datagrid('updateRow', {
								index: vm.addQosForm.INDEX,
								row: parmas
							});
						}
						
					}
					delete parmas._edit;
					delete parmas._remove;
					toTbDataQueue(qosTb, parmas, idField);
					openPropsPanel(false);
				}else{
					return false
				}
			} );
		},
		// 取消新增
		closeAdd(){
			openPropsPanel(false);
		},
		getTFTAndQoSTableList(){
			var vm = this,precedenceList=[],pccNameList = [],pccIdList=[];
			vm.QoSList = $(qosTb).datagrid('getRows');
			vm.TFTList = $(tftTb).datagrid('getRows');
			vm.QoSList.map((item)=>{
				if(item["56C85E6D9E1F0229854130A5FAA47FFC"] !== vm.rowData["56C85E6D9E1F0229854130A5FAA47FFC"]){
					precedenceList.push(parseInt(item["5540762DDECD826BBC6AED827744F0A2"]));
					pccNameList.push(item["56C85E6D9E1F0229854130A5FAA47FFC"]);
					pccIdList.push(item["D303A3555BED53E8FC5F34E883104904"]);
				}
				if(item["5A5BA201950EDCAB6F15E29DA226185A"]){
					var item_pfIdList = item["5A5BA201950EDCAB6F15E29DA226185A"].split(',');
					item_pfIdList.map((items)=>{
						vm.selectedPfList.push(items);
					})
				}
			})
			var newTFTList=[];
			newTFTList = vm.TFTList.filter(item=>!vm.selectedPfList.includes(item["BC3DC315DF2D8654F8995B0D8696C800"]))
			vm.precedenceList = precedenceList;
			vm.pccNameList = pccNameList;
			if(vm.operType == 'add'){
				var tftTableList ={
					left:newTFTList,
					right:[]
				}
				vm.tftTableList = tftTableList;
			}else{
				vm.old_PCC_ID = vm.rowData["D303A3555BED53E8FC5F34E883104904"]||'';
				vm.addQosForm.PCC_NAME = vm.rowData["56C85E6D9E1F0229854130A5FAA47FFC"]||'';
				vm.addQosForm.PCC_ID = vm.rowData["D303A3555BED53E8FC5F34E883104904"]||'';
				vm.addQosForm.INDEX = $(qosTb).datagrid('getRowIndex',vm.rowData);
				vm.addQosForm.QCI = vm.rowData["BE5E24E4B68D8037D68B7047C13C0A93"];
				vm.addQosForm.MBR_UL = vm.rowData["702A697631DA87471CFBF0F6AE1B2DDA"]||'';
				vm.addQosForm.MBR_DL = vm.rowData["A175EC59C27D988CED837D8A31D1AFD6"]||'';
				vm.addQosForm.GRB_UL = vm.rowData["D4BD7B7FBCE7FE09C76FF9B2B0B63DF9"]||'';
				vm.addQosForm.GRB_DL = vm.rowData["0EAB6B40D32DBF65DE0A93345B517E01"]||'';
				vm.addQosForm.ARP_PL = vm.rowData["BB88A1C448BE32514BD74C8F6296B8F1"]||'';
				vm.addQosForm.ARP_PCI = vm.rowData["FA7036C847AD76E239E9A0538CA0EE37"]||'';
				vm.addQosForm.ARP_PVI = vm.rowData["D4C8710204B6D3D3EA5DF0D34C36B3E6"]||'';
				vm.addQosForm.PRECEDENCE = vm.rowData["B4658365E4D73CD935B128E64F14F6AB"]||'';
				vm.addQosForm.PF_List = vm.rowData["5A5BA201950EDCAB6F15E29DA226185A"]||'';
				vm.addQosForm.cellCodes = vm.rowData["5A5BA201950EDCAB6F15E29DA226185A"]||'';

				var tftTableList = {
						left:[],
						right:[]
					};
				if(vm.addQosForm.PF_List == '' || vm.addQosForm.PF_List == null){

				}else{
					var pfIdList = vm.addQosForm.PF_List.split(',');
					pfIdList.map((item)=>{
						vm.TFTList.map((its)=>{
							if(item == its["BC3DC315DF2D8654F8995B0D8696C800"]){
								tftTableList.right.push(its);
								newTFTList.push(its);
							}
						})
					})
				}
				tftTableList.left = newTFTList;
				vm.tftTableList = tftTableList;
			}
		},
		isNumeric(str) {
			if(str.length==0){
				return false;
			}
			for(var i=0;i<str.length;i++){
				if(str.charAt(i)<"0" || str.charAt(i)>"9"){
					return false;
				}
			}
			return true;  
		},
		// 查看是否有修改或删除的数据  修改状态
		setValidResult(){
			var vm = this;
			if(Render.tableCollector['17EF1A3DD71826B1E0DCED9FDDA82005']){
				Render.tableCollector['17EF1A3DD71826B1E0DCED9FDDA82005'].map((items,index)=>{
					if(items.operateType !== 'add'){
						Render.validResult.reboot = true;
					}
				})
			}
			if(Render.tableCollector['C5F614F36FAFBAA281777B754312924D']){
				Render.tableCollector['C5F614F36FAFBAA281777B754312924D'].map((items,index)=>{
					if(items.operateType !== 'add'){
						Render.validResult.reboot = true;
					}
				})
			}
			
		}
	},
	mounted(){
		
        eventBus.$off('qos-init').$on('qos-init',this.init);
	}
	
})

</script> 
