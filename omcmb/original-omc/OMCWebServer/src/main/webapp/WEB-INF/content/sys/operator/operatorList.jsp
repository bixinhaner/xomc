<%@ page import="java.util.Map" %>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style type="text/css">
.pageContainer{
	display:flex;
	height:100%;
}
.operatorCon{
	flex:1 1 37%;
	background:#FFF;
}
.operatTableList{
	flex: 1 1 63%;
	background:#ffffff;
	border-left:2px solid #e9e9e9;
	box-sizing:norder-box;
}
.tableHeader{
	display:flex;
	box-sizing:border-box;
	position:relative;
	padding:0px 20px;
	align-items:center;
}
.el-slide{
	right:0px !important;
}
.searchCon{
	width:400px;
	margin-left:27px;
}
.searchCon .el-input{
	width:100%;
}
.searchCon .el-input__inner{
	border-radius:4px;
	background:#FFFFFF;
	border: 1px solid #E9E9E9;
	height:30px;
	line-height:30px;
}
.buttonGroup{
	position:absolute;
	right:20px;
	top:0px; 
	z-index:10;
}
.buttonGroup div{
	width:36px;
	float:left;
}
.buttonGroup p{
	text-align:center;
	font-size:16px;
	color:#1DA3FC;
	margin-top:5px;
}
.buttonGroup .el-button:focus,.buttonGroup .el-button:hover{
	background-color:#1DA3FC;
}
.headerTitle{
	font-weight:bold;
	color:#363B4E;
	font-size:16px;
}
.operatorCon .el-dialog{
	width:22%;
}
#operatorPage .el-tabs{
	height:100%;
}
.select-device-container{
	position:absolute;
	height:60px;
	background:rgba(255,255,255,1);
	box-shadow:0px -5px 10px rgba(0,0,0,0.1);
	bottom:0px;
	left:0px;
	right:0px;
	box-sizing:border-box;
	padding:0px 20px 0px 20px;
	display:flex;
	align-items:center;
	justify-content:space-between;
	z-index:1;
}
.titleSpan{
	font-weight:bold;
	color:#333333;
	font-size:14px;
	align-items:center;
}
.selfbutton{
	height:24px;
	line-height:24px;
	padding:0px 20px;
	margin-right:15px;
}
.operatTableList .el-tabs__nav{
	margin-left:18px;
}
.operatorCon .el-message-box__content, .operatorCon .el-dialog__body{
	padding:30px;
}
#operatorPage .el-dialog__footer{
	padding:0 30px 10px;
	display:flex;
}
#operatorPage .deldialog .el-dialog__footer{
	display:block;
	margin-right:20px;
}
.addOperDia .el-input{
	width:300px;
}
#operatorPage .el-tabs__nav-wrap{
	border-bottom:1px solid #E9E9E9;
}
.deldialog .el-form-item{
	display:flex;
	align-items:center;
	margin-top:15px;
	margin-bottom:0px;
}
.has-select-container{
	position:absolute;
	width:380px;
	height:400px;
	box-sizing:border-box;
	display:flex;
	flex-direction:column;
	bottom:60px;
	left:0px;
	background:#FFFFFF;
	box-shadow:5px -5px 15px rgba(0,0,0,0.1)
}
.select-container-header{
	height:40px;
	border-bottom:1px solid #E0E4ED;
	padding:0px 20px;
	box-sizing:border-box;
	display:flex;
	line-height:40px;
	justify-content: space-between;
}
.border-header-tltle{
	font-weight:bold;
	color: #363B4E;
	font-size: 16px;
}
.select-container-body{
	flex: 1;
	overflow:auto;
	display: flex;
	flex-direction: column;
	box-sizing: border-box;
	padding: 0px 10px;
}
.select-container-body div{
	padding: 0px 10px;
	box-sizing: border-box;
}
.select-body-header{
	height: 46px;
	border-bottom:1px solid #F4F4F4;
	display: flex;
	justify-content: space-between;
	line-height: 46px;

}
.body-header-title{
	font-weight: bold;
	color: #333333;
	font-size: 14px;
}
.body-list{
	flex: 1;
	overflow: auto;
}
.select-body-list{
	height: 36px;
	line-height:36px;
	display: flex;
	justify-content: space-between;
	border-bottom: 1px solid #F4F4F4;
}
.select-body-list i{
	visibility: hidden;
}
.select-body-list:hover i{
	visibility: visible;
}
.select-body-list:nth-child(2n+1){
	background: #FCFDFF;
}
#operatorPage .el-input__suffix{
	top:4px;
}
</style>
<div id="operatorPage" class="pageContainer" style="overflow: auto;">
	<!-- 运行商列表 -->
	<div class=" operatorCon" style="overflow: auto;">
		<el-ctable ref="operListTable" @row-click="selectOperator" :url="operListTableUrl" :query-params="params" :height="height" pagination="true" :rownumber="rownumber">
			<template slot="toolbar">
				<div class="tableHeader">
					<span class="headerTitle"><%=rb.getString("YunYingShang") %></span>
					<div class="searchCon" >
						<el-input placeholder='<%=rb.getString("YunYingShangMingCheng") %> / <%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("CPEMacAddress")%>' v-model="operListSearchText">
							<i slot="suffix" class="el-icon el-icon-common-search" @click="refreshOperListTable"></i>
						</el-input>
					</div>
					<div v-if="false" class='buttonGroup' >
						<temp-button title='<%=rb.getString("TianJia") %>' :temp-class='importClass' @temp-click="dialogOperatorFormVisible = true"></temp-button>
					</div>
				</div>
				
				<!-- 添加运营商弹出框 -->
				<el-dialog title="New Operator" class="addOperDia" :close-on-click-modal=false :visible.sync="dialogOperatorFormVisible">
					<el-form ref="operatorAddForm" :model="form" :rules="rules" label-position="top">
						<el-form-item label="<%=rb.getString("YunYingShangMingCheng")%><%=rb.getString("MaoHao")%>" prop="operatorCode">
							<el-input :disabled="disableFlag" @blur="fillDefaultAdmin" v-model="form.operatorCode"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("CLOUDKEY")%><%=rb.getString("MaoHao")%>" prop="cloudKey">
							<el-input :disabled="disableFlag" @focus="fillDefaultCloudKey" v-model="form.cloudKey"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("MoRenGuanLiYuan")%><%=rb.getString("MaoHao")%>" prop="adminUserCode">
							<el-input :disabled="disableFlag" v-model="form.adminUserCode"></el-input>
						</el-form-item>
					</el-form>
					<div slot="footer" class="dialog-footer">
						<el-button type="primary" @click="addOperator"><%=rb.getString("QueDing")%></el-button>
						<el-button @click="dialogOperatorFormVisible = false"><%=rb.getString("QuXiao")%></el-button>
					</div>
				</el-dialog>
				
			
			</template>
			<el-table-column prop="operation" label="" width="30" v-if="!hideOperation" class-name="operationColumn">
				<template slot-scope="scope">
            				<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
          			</template>
			</el-table-column>
			<el-table-column prop="operator_name" width="120" label="<%=rb.getString("YunYingShangMingCheng")%>" ></el-table-column>
			<el-table-column prop="cloud_key" label="<%=rb.getString("CLOUDKEY")%>" ></el-table-column>
			<el-table-column prop="eNodeBCount" label="<%=rb.getString("XiaoZhan")%>" ></el-table-column>
			<el-table-column prop="CPECount" label="<%=rb.getString("CPE")%>" ></el-table-column>
			<el-table-column prop="is_beta" label="Beta" sortable>
				<template  slot-scope="scope">
						<div class="tableTdContainer" v-if="scope.row.statisticType == 0">
							<span><%=rb.getString("SheBeiTongJi")%></span>
						</div>
						<div  v-if="scope.row.is_beta == '0'"><%=rb.getString("Fou")%></div>
						<div v-else ><%=rb.getString("Shi")%></div>
					</template>
			</el-table-column>
		</el-ctable>
		<el-cmenu ref="operListMenu" @click="clickMenu" :data="menus"></el-cmenu>
		
	</div>
	<!-- eNB cpe列表 -->
	<div class="operatTableList" style="overflow: auto;">
		<template>
			<el-tabs v-model="deviceTypeTabs" @tab-click="changeDeviceTable">
				<el-tab-pane label="eNB" name="enb" style="overflow:auto">
					<el-ctable @selection-change="changeEnbSelect" ref="deviceEnbTable" :url="deviceEnbTableUrl" :query-params="paramsEnb" :height="height" pagination="true" :rownumber="rownumber">
						<template slot="toolbar">
							<div class="searchCon" style="margin-left:18px;">
								<el-input placeholder="<%=rb.getString("XiaoZhanBianMa")%>" v-model="enbSearchText">
									<i slot="suffix" class="el-icon el-icon-common-search" @click="refreshEnbTable"></i>
								</el-input>
								
							</div>
						</template>
						<el-table-column type="selection" width="55" ></el-table-column>
						<el-table-column prop="serial_number" label="<%=rb.getString("XiaoZhanBianMa")%>" ></el-table-column>
					</el-ctable>
				</el-tab-pane>
				<el-tab-pane label="CPE" name="cpe" style="overflow:auto">
					<el-ctable @selection-change="changeCpeSelect" ref="deviceCpeTable" :url="deviceCpeTableUrl" :query-params="paramsCpe" :height="height" pagination="true" :rownumber="rownumber">
						<template slot="toolbar">
							<div class="searchCon" style="margin-left:18px;">
								<el-input placeholder="<%=rb.getString("CPEXuLieHao")%>" suffic-icon='el-icon-search' v-model="cpeSearchText">
									<i slot="suffix" class="el-icon el-icon-common-search"  @click="refreshCpeTable"></i>
								</el-input>
							</div>
						</template>
						<el-table-column type="selection" width="55" ></el-table-column>
						<el-table-column prop="serial_number" label="<%=rb.getString("CPEXuLieHao")%>" ></el-table-column>
						<el-table-column prop="macaddress" label="<%=rb.getString("CPEMacAddress")%>" ></el-table-column>
					</el-ctable>
				</el-tab-pane>
			</el-tabs>
			<!-- @load-success="loadOperSuccess" --><!--:data="operatorSelectTableData" :url="operatorSelectTableUrl"-->
			<el-dialog title="<%=rb.getString("XuanZeYunYingShang")%>" :close-on-click-modal=false :visible.sync="dialogselectOperatorFormVisible" @close="cancelMoveTooperator">
				<el-ctable border=true ref="changeOperaTable"  @row-click="rowClickUpgrade" :url="operatorSelectTableUrl" :query-params="paramsoper" :height="operheight" pagination="true" :rownumber="rownumber">
					<template slot="toolbar">
						<div class="searchCon" style="margin-left:0px;">
							<el-input placeholder="<%=rb.getString("YunYingShangMingCheng")%>" v-model="operSearchText">
								<i slot="suffix" class="el-icon el-icon-common-search" @click="refreshSelectOper"></i>
							</el-input>
						</div>
					</template>
					<el-table-column label='<%=rb.getString("XuanZe")%>' width="80">
					<template slot-scope="scope">
		              		<div class='tableDiv el-icon el-icon-status-yes selected-status' style="cursor: pointer;text-align:center;line-height:23px;"></div>
		            	</template>
					</el-table-column>
					<el-table-column prop="operator_name" label="<%=rb.getString("YunYingShangMingCheng")%>" ></el-table-column>
					<el-table-column prop="cloud_key" label="<%=rb.getString("CLOUDKEY")%>" ></el-table-column>
				</el-ctable>
				<div slot="footer" class="dialog-footer">
					<el-button type="primary" @click="saveMoveTooperator"><%=rb.getString("QueDing")%></el-button>
					<el-button style='margin-left:10px;' @click="cancelMoveTooperator"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</el-dialog>
		</template>
	</div>
	<!-- 设备选择 -->
	<div class="select-device-container" v-if="showflag">
		<div style="height:24px;"><span class="titleSpan" style="margin-left:10px;margin-right:10px;"><%=rb.getString("YiXuanSheBei")%></span>(<span style="color:#4D84FF">{{selectDeviceNum}}</span>)<i style="font-size:14px;margin-left:10px;" :class="showSelectDeviceFlag ? 'el-icon-circle-down' : 'el-icon-circle-up'" class="el-icon el-icon-circle-down" @click="showSelectDeviceFlag = !showSelectDeviceFlag"></i></div>
		<div class="">
			<el-button type="primary" v-show="moveShow" class="selfbutton" @click="moveOperator"><%=rb.getString("YiDongDaoYunYingShang")%></el-button>
			<el-button style="margin-right:0px;" class="selfbutton" class="selfbutton" @click="changeDeviceTable"><%=rb.getString("QuXiao")%></el-button>
		</div>
		
			<!-- 选择设备详情列表 -->
		<transition name="slide">	
			<div v-if="showSelectDeviceFlag" class="has-select-container">
				<div class="select-container-header">
					<span class="border-header-tltle">Selected Devices</span>		
					<i class="el-icon el-icon-close" @click="showSelectDeviceFlag = !showSelectDeviceFlag" style='position:absolute;right:10px;top:15px;'></i>		
				</div>
				<div class="select-container-body">
					<div class="select-body-header">
						<span v-if="deviceTypeTabs == 'enb' " class="body-header-title"><%=rb.getString("XiaoZhanBianMa")%></span>
						<span v-else class="body-header-title"><%=rb.getString("CPEXuLieHao")%></span>
						<span><i class="el-icon el-icon-operation-delete" @click="changeDeviceTable"></i><span style="font-size:12px;color:#333333;font-family:PingFang SC;font-weight:bold;margin-left:2px;"><%=rb.getString("QingChu")%></span></span>
					</div>
					<div class="body-list" v-if="deviceTypeTabs == 'enb' ">
						<div v-for="(item,index) in hasSelectDevice" :key="index" class="select-body-list">
							<span>{{item.serial_number}}</span><i class="el-icon el-icon-circle-close" style='font-size:18px;margin-top:10px;' @click="cancelSelectDevide(item.small_cell_code)"></i>
						</div>
					</div>
					<div class="body-list" v-else>
						<div v-for="(item,index) in hasSelectDevice" :key="index" class="select-body-list">
							<span>{{item.serial_number}}</span><i class="el-icon el-icon-circle-close" style='font-size:18px;margin-top:10px;' @click="cancelSelectDevide(item.cpe_code)"></i>
						</div>
					</div>
				</div>
			</div>
		</transition>
	</div>
	
	
	
	<el-slide ref="viewoperatorSlide" :url='viewSlideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition" class="tableContent"
	:height="slideHeight" :modal='modal' :width='slideWidth' @ok="saveSlide" @cancel="cancelSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-slide>
	<!-- 删除运营商弹出框 -->
	<el-dialog class="deldialog" title="Confirm" width="25%" :close-on-click-modal=false :visible.sync="dialogdelFormVisible">
		<span><%=rb.getString("QueRenShanChuYunYingShang")%></span>
		<el-form ref="delForm" :model="delForm" :rules="delRules">
			<el-form-item label="Password" prop="delPassword">
				<el-input type="password" v-model="delForm.delPassword" auto-complete="new-password" placeholder="<%=rb.getString("QingShuRuMiMa")%>"></el-input>
			</el-form-item>
				<el-input type="password" v-show=false placeholder="<%=rb.getString("QingShuRuMiMa")%>"></el-input>
		</el-form>
			<div slot="footer" class="dialog-footer" style="display:unset;">
				<el-button type="primary" @click="suerDel"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="dialogdelFormVisible = false"><%=rb.getString("QuXiao")%></el-button>
			</div>
	</el-dialog>
	
</div>

<template id='elCardTopButton'>
	<div>
		<i :class='tempClass' @click="$emit('temp-click')" @mouseover.native='showText' @mouseout.native='hideText'></i>
		<p v-show='operText'>{{title}}</p>
	</div>
</template>
<script type="text/javascript">
var eventBus = new Vue();
	new Vue({
		el:'#operatorPage',
		data(){
			var validateDefaultAdmin = (rule,value,callback)=>{
				var reg = /^[a-zA-Z0-9\_\-\@\.]+$/;
				//必填验证
				if(!value){
					callback(new Error('<%=rb.getString("QingShuRuYunYingShangMingCheng")%>'));
				}else{
					if(reg.test(value)){
						callback()
					}else{
						callback(new Error("<%=rb.getString("YunYingShangChengBuHeFa")%>"));
					}
				}
			}
			var validateCloudKey = (rule,value,callback)=> {
				var reg = /^[0-9A-Z]{6}$/;
				if(reg.test(value)){
					callback();
				}else{
					callback(new Error("<%=rb.getString("CLOUDKEYYOUXIAOXING")%>"));
				}
			}
			return {
				showSelectDeviceFlag:false,
				hasSelectDevice:[],
				delOperatorCode:'',
				viewSlideUrl:'',
				slideTitle:'',
				slideFooter:'',
				slideHeader:'',
				slidePosition:'',
				slideHeight:'',
				slideWidth:'',
				modal:'',
				showflag:false,
				selectDeviceNum:0,
				deviceTypeTabs:'enb',
				disableFlag:false,
				enbSearchText:'',
				cpeSearchText:'',
				operSearchText:'',
				form:{
					operatorCode:'',
					cloudKey:'',
					operatorName:'',
					adminUserCode:''
				},
				delForm:{
					delPassword:'',
				},
				dialogOperatorFormVisible:false,
				dialogselectOperatorFormVisible:false,
				dialogdelFormVisible:false,
				searchText:'',
				operListSearchText:'',
				params:{
					searchText:'',
				
				},
				operator_code_current:'default',
				paramsEnb:{
					search_text:'',
					operator_code:'default',
					like_fields:'serial_number'
				},
				paramsCpe:{
					search_text:'',	
					operator_code:'default',
					like_fields:'serial_number'
				},
				paramsoper:{
					operator_code:'',
					operator_code_current:'default'
					//like_fields:'serial_number'
				},
				importClass:'el-icon-circle-add el-icon',
				operListTableUrl:'${ctx}/system/operator/getOperatorListAndCounts.action',
				deviceEnbTableUrl:'${ctx}/system/device/enodeb/queryENBInfoPageList.action',
				deviceCpeTableUrl:'${ctx}/system/device/cpe/queryCPEInfoPageList.action',
				operatorSelectTableUrl:'${ctx}/system/operator/getOperatorListExceptself.action?no_built_in=1',
				/* operatorSelectTableData:[
					{
						operator_name: "11",
						cloud_key: "36VZ1T",
						
					},
					{
						operator_name: "1222345",
						cloud_key: "36VZ1T",
						
					},
					{
						operator_name: "5G",
						cloud_key: "36VZ1T",
						
					}
				], */
				height:'100%',
				operheight:'40%',
				pageSize:50,
				rownumber:true,
				menus:[],
				enbIDs:'',
				cpeCodes:'',
				operatorRow:'',
				delRules:{
					delPassword:{required:true,message:"<%=rb.getString("QingShuRuMiMa")%>",trigger:'blur'}
				},
				rules:{
					operatorCode:[
						{validator:validateDefaultAdmin}
					],
					cloudKey:[
						{validator:validateCloudKey}
					],
					adminUserCode:[
						{require:true,message:'<%=rb.getString("QingShuRuYunYingShangMingCheng")%>'}
					]
				},
				hideOperation:isCloudCore == "true" ? true : false
			}
		},
		computed: {
			moveShow() {
				return writableMap['CODE_OPERATOR_MANAGEMENT'] == true;
			}
		},
		methods:{
			/* loadOperSuccess(){
				
			}, */
			selectOperator(row,column,event){
				this.paramsCpe.operator_code = row.operator_code;
				this.paramsEnb.operator_code = row.operator_code;
				this.paramsoper.operator_code_current = row.operator_code;
				this.operator_code_current = row.operator_code;
				this.refreshEnbTable();
				this.refreshCpeTable();
				//this.refreshSelectOper();
			},
			changeEnbSelect(selection){
				var vm = this;
				vm.hasSelectDevice = [];
				vm.hasSelectDevice = selection;
				vm.selectDeviceNum = selection.length;
				if(vm.selectDeviceNum != 0){
					vm.showflag = true;
				}else{
					this.showSelectDeviceFlag = false;
					vm.showflag = false;

				}
				var ids = "";
				for (var i = 0; i < selection.length; i++) {
					ids += selection[i].small_cell_code + "_" + selection[i].product + ",";
				}
				vm.enbIDs = ids;
				
			},
			changeCpeSelect(selection){
				var vm = this;
				vm.hasSelectDevice = [];
				vm.hasSelectDevice = selection;
				vm.selectDeviceNum = selection.length;
				if(vm.selectDeviceNum != 0){
					vm.showflag = true;
				}else{
					this.showSelectDeviceFlag = false;
					vm.showflag = false;
					
				}
				var cpeCodes = "";
				for (var i = 0; i < selection.length; i++) {
					cpeCodes += selection[i].cpe_code + ",";
				}
				vm.cpeCodes = cpeCodes;
				
			},
			changeDeviceTable(){
				//清空选择项
				var vm = this;
				this.showSelectDeviceFlag = false;
				vm.showflag = false;
				if(this.deviceTypeTabs == "enb"){
					vm.$refs["deviceEnbTable"].clearSelection();
				}else{
					vm.$refs["deviceCpeTable"].clearSelection();
				}
				
			},
			cancelSelectDevide(smallCellCode){
				var vm = this;
				let tableData = null;
				let rowIndex = 0;
				if(this.deviceTypeTabs == "enb"){
					tableData = vm.$refs["deviceEnbTable"].getData();
					try { //forEach不能break 跳出 只能trycath 跳出整个循环
						tableData.forEach((item,index) => {
							if(item.small_cell_code == smallCellCode){
								rowIndex = index;
								throw new Error()
							}
						})
					} catch (err) {}
				}else{
					tableData = vm.$refs["deviceCpeTable"].getData();
					try { //forEach不能break 跳出 只能trycath 跳出整个循环
						tableData.forEach((item,index) => {
							if(item.cpe_code == smallCellCode){
								rowIndex = index;
								throw new Error()
							}
						})
					} catch (err) {}
				}
				if(this.deviceTypeTabs == "enb"){
					tableData = vm.$refs["deviceEnbTable"].toggleRowSelection(tableData[rowIndex],false);
				}else{
					tableData = vm.$refs["deviceCpeTable"].toggleRowSelection(tableData[rowIndex],false);
				}
			},
			moveOperator(){
				var vm = this;
				if(vm.deviceTypeTabs == "enb"){
					if(vm.enbIDs == ""){
						vm.$message('<%=rb.getString("QingXuanZeSheBei")%>')
					}
				}else{
					if(vm.cpeCodes == ""){
						vm.$message('<%=rb.getString("QingXuanZeSheBei")%>')
					}
				}
				//vm.operSearchText = ''; //清空搜索项
				//vm.paramsoper.operator_code = ''; //搜索参数清空
				vm.dialogselectOperatorFormVisible = true;
			},
			// select operator 确定事件
			saveMoveTooperator(){
				var vm = this;
				//判断是否选择了
				if(!this.operatorRow){
					vm.$message.error("<%=rb.getString("QingXuanZeYunYingShang")%>") //错误提示信息
					return false;
				}
				var params = {};
				var url = ""
				if(this.deviceTypeTabs == "enb"){
					params.ids = this.enbIDs;
					url="${ctx}/system/deviceGroup/moveCellToOperator.action";
				}else{
					params.cpeCodes = this.cpeCodes;
					url="${ctx}/cell/CPE/moveCpeToOperator.action";
				}
				params.toOperatorCode = this.operatorRow.operator_code;
				axios.post(url,stringify(params)).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
						this.showSelectDeviceFlag = false;
		    			vm.showflag = false;
		    			if(vm.deviceTypeTabs == "enb"){
		    				vm.$refs["deviceEnbTable"].refresh();
		    			}else{
		    				vm.$refs["deviceCpeTable"].refresh();
		    			}
		    			vm.$refs["operListTable"].refresh();
		    		
		    			vm.dialogselectOperatorFormVisible = false;
		    			vm.$refs.changeOperaTable.refresh();
		    			vm.$refs.changeOperaTable.setCurrentRow();
		    			vm.operatorRow = '';
		    			vm.operSearchText = ''; //清空搜索项
						vm.paramsoper.operator_code = ''; //搜索参数清空
		    			vm.$message({
		    				type:'success',
		    				message:"<%=rb.getString("ChengGong")%>"
		    			})
		    		}else{
		    			vm.$message.error(data["message"]) //错误提示信息
		    		}
		    	})
			},
			//select operator 取消事件  取消被选中的数据
			cancelMoveTooperator(){
				this.dialogselectOperatorFormVisible = false;
    			this.$refs.changeOperaTable.refresh();
    			this.operatorRow = '';
    			this.operSearchText = ''; //清空搜索项
    			this.paramsoper.operator_code = ''; //搜索参数清空
    			this.$refs.changeOperaTable.setCurrentRow();
			},
			refreshOperListTable(){
				var vm = this;
				vm.params.searchText = vm.operListSearchText;
				vm.$refs["operListTable"].refresh();
			},
			refreshEnbTable(){
				var vm = this;
				vm.paramsEnb.search_text = vm.enbSearchText;
				vm.$refs["deviceEnbTable"].refresh();
			},
			refreshCpeTable(){
				var vm = this;
				vm.paramsCpe.search_text = vm.cpeSearchText;
				vm.$refs["deviceCpeTable"].refresh();
			},
			refreshSelectOper(){
				var vm = this;
				vm.paramsoper.operator_code = vm.operSearchText;
				vm.paramsoper.operator_code_current = vm.operator_code_current
				vm.$refs["changeOperaTable"].refresh();
			},
			suerDel(){
				var vm = this;
				var params = {"operatorCode": this.delOperatorCode,adminPassword:this.delForm.delPassword};
				this.$refs["delForm"].validate((valid)=>{
					 if(valid){
						 axios.post('${ctx}/system/operator/delOperator.action',stringify(params)).then(function(response){
					    		var data = response.data;
					    		if(data["success"]){
					    		
					    			vm.$refs["operListTable"].refresh();
					    			if(vm.delOperatorCode == operator_code){
					    				setCurrOperator("default");
					    			}
					    			vm.$refs["delForm"].resetFields();
					    			vm.dialogdelFormVisible = false;
					    		}else{
					    			vm.$message.error("<%=rb.getString("ShanChuYunYingShangShiBai")%>") //错误提示信息
					    		}
					    	})
					 }
				})
			},
			
			optClick(row,ev){
		    	var vm = this;
		    	this.rowData = row;
		    	var disableFlag = null;
		    	if(row.built_in != '1'){
		    		disableFlag = false;
		    	} else{
		    		disableFlag = true;
		    	}
		    	vm.menus = [
					{label:'<%=rb.getString("XinXi")%>',cls:'el-icon-operation-info el-icon',code:'info'},
					{label:'<%=rb.getString("XiuGai")%>',cls:'el-icon-operation-edit el-icon CODE_OPERATOR_MANAGEMENT hidden',code:'edit'},
					{label:'<%=rb.getString("ShanChu")%>',cls:'el-icon-operation-delete el-icon CODE_OPERATOR_MANAGEMENT hidden',code:'del',disable:disableFlag}
				]
		    	this.$nextTick(()=>{
					document.body.click();
					vm.$refs.operListMenu.show(ev)
				})
		    },
		    clickMenu(ev){ //菜单点击对应的方法
				var codes = {
					info:this.infoOperList,
					del:this.delOperList,
					edit:this.editOperList,
				};
				//根据code 判断执行哪个方法
				codes[ev.code](this.$root.rowData["operator_code"]);
				/* if(ev.code == "view"){
					codes[ev.code](this.$root.rowData["statisticId"],this.$root.rowData["statisticTime"],this.$root.rowData["startTime"],this.$root.rowData["endTime"],this.$root.rowData["statisticType"],this.$root.rowData["statisticName"]);
				}else{
					codes[ev.code](this.$root.rowData["statisticId"]);
				} */
		  },
		  infoOperList(operatorCode){
			  var vm = this;
			  vm.slideHeader = true;
	      	  vm.slideTitle = '<%=rb.getString("XinXi")%>';
	  	      vm.viewSlideUrl = '${ctx}/system/operator/toView.action';
	  	      vm.slideFooter = false;
	  	      vm.slidePosition = 'right';
	  	      vm.slideHeight = '100%';
	  	      vm.slideWidth = '65%';
	  		  vm.$refs.viewoperatorSlide.showSlide(()=>{
	  		 	vm.model = true;
	  			eventBus.$emit('info-task',operatorCode,'view')
	  		 })
		  },
		  editOperList(operatorCode){
			  var vm = this;
			  vm.slideHeader = true;
	      	  vm.slideTitle = '<%=rb.getString("XiuGai")%>';
	  	      vm.viewSlideUrl = '${ctx}/system/operator/toView.action';
	  	      vm.slideFooter = true;
	  	      vm.slidePosition = 'right';
	  	      vm.slideHeight = '100%';
	  	      vm.slideWidth = '65%';
	  		  vm.$refs.viewoperatorSlide.showSlide(()=>{
	  		 	vm.model = true;
	  			eventBus.$emit('info-task',operatorCode,'modify')
	  		 })
		  },
		  saveSlide(){
			  eventBus.$emit('save-edit-task')
		  },
		  cancelSlide(){
			  var vm = this;
			  vm.$refs.viewoperatorSlide.hide();
			  vm.$refs["operListTable"].refresh();
		  },
		  delOperList(operatorCode){
			  this.dialogdelFormVisible = true;
			  this.delOperatorCode = operatorCode;
			  eventBus.$emit("del-operator",operatorCode)
		  },
		  rowClickUpgrade(row){
	        this.operatorRow = row;
	       },
		  handerClose(){
		  	this.$refs.operListMenu.hide();
		  },
		  //添加运营商相关的函数
		  //随机生成CLOUDKEY
		 fillDefaultCloudKey(){
			if(this.form.cloudKey == ""){
				this.form.cloudKey = Math.random().toString(30).substring(5).slice(0,6).toUpperCase();
			}
		 },
		 fillDefaultAdmin(){
			this.form.adminUserCode = this.form.operatorCode + "Admin"
			
		 },
		 addOperator(){
			 var vm = this;
			 this.$refs["operatorAddForm"].validate((valid)=>{
				 if(valid){
					 var params = {};
					 Object.assign(params,vm.form);
					 axios.post('${ctx}/system/operator/addOperator.action',stringify(params)).then(function(response){
			    		var data = response.data;
			    		if(data["success"]){
			    			vm.dialogOperatorFormVisible = false;
			    			vm.$refs["operListTable"].refresh();
			    			vm.$refs["operatorAddForm"].resetFields();
			    		}else{
			    			vm.$message.error(data["message"]) //错误提示信息
			    		}
			    	})
				 }
			 })
		 }
		 
		},
		mounted(){
			eventBus.$off("close-slide").$on("close-slide",this.cancelSlide);

		},
		components:{
			'temp-button':{
				template:'#elCardTopButton',
				data(){
					return{
						operText:false
					}
				},
				props:['title','tempClass'],
				methods:{
					showText(){
						this.operText = true;
					},
					hideText(){
						this.operText = false;
					},
					
				}
			},
		},
	
	})
</script>