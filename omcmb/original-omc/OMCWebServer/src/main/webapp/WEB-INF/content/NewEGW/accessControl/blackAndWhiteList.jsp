<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#egwBlackAndWhiteList .container .operations{
		right: 10px;
	}
	#egwBlackAndWhiteList .slide-position-top .el-card__body{
		box-sizing: border-box !important;
	}
	#egwBlackAndWhiteList .blackAndWhiteListMainBox{
		height: 100%;
		display: flex;
		justify-content:space-between;
	}
	#egwBlackAndWhiteList .container .cmenu{
		z-index: 361!important;
	}
	#egwBlackAndWhiteList .headBtnCls .el-icon::before{
		font-size: 30px;
	}
	#egwBlackAndWhiteList .list_left_box{
		height: 100%;
		flex: 3;
		min-width: 480px;
		position: relative;
	}
	#egwBlackAndWhiteList .list_right_box{
		height: 100%;
		flex: 5;
		border-left:1px solid #E9E9E9;
		position: relative;
	}
	#egwBlackAndWhiteList .blackListTable,#egwBlackAndWhiteList .whiteListTable{
		height: calc(50% - 25px);
		width: 100%;
		position: relative;
	}
	#egwBlackAndWhiteList .whiteListTable{
		border-bottom:1px solid #E9E9E9;
	}
	#egwBlackAndWhiteList .activeItemCls .el-icon::before{
		font-size: 16px;
		color: #67D972;
	}
	#egwBlackAndWhiteList .noActiveItemCls .el-icon::before{
		font-size: 16px;
		color: #e88282;
	}
	#egwBlackAndWhiteList div{
		box-sizing: border-box;
	}
	#egwBlackAndWhiteList .list_right_box{
		display: flex;
		flex-direction: column;
	}
	#egwBlackAndWhiteList .right_box_header{
		height: 50px;
		border-bottom: 1px solid #E9E9E9;
		display: flex;
		align-items: center;
	}
	#egwBlackAndWhiteList .promptInfoCls{
		margin-left: 10px;
		font-size: 12px;
		font-weight: bold;
		color:#363B4E;
		margin-right: 60px;
	}
	#egwBlackAndWhiteList .modeLabelCls{
		margin-left:60px;
		margin-right:10px;
		color:#363B4E;
		font-weight: bold;
	}
	#egwBlackAndWhiteList .right_box_header .el-select>.el-input,
	#egwBlackAndWhiteList .right_box_header .el-select .el-input .el-input__inner{
		width: 100px !important;
	}
	#egwBlackAndWhiteList .headQueryBox{
		display: flex;
		align-items: center;
	}
	#egwBlackAndWhiteList .headQueryBox .el-input.el-input--small{
		width: 300px;
	}
	#egwBlackAndWhiteList .el-form-item__error{
		padding-top: 0px;
	}
	.el-input-group__append{
		border-radius:0px;
		border-right:none;
	}
	.validate-item .el-input__inner{
		width:150px;
	}
	.validate-item .el-input-group__append{
		border:none;
		background:none;
	}
	.validate-item .el-form-item__error{
		display:none;
	}
	.is-error .el-input-group__append{
		color:#FA5555;
	}
	.addWhiteBlackDialogCls .itemListBoxCls{
		padding-left: 140px;
	}
	.addWhiteBlackDialogCls .itemCls{
		height: 24px;
		display: inline-block;
		line-height: 24px;
		border: 1px solid #4D84FF;
		box-sizing: border-box;
		padding: 0px 10px;
		margin-right: 10px;
		margin-bottom: 10px;
		overflow: hidden;
	}
	.addWhiteBlackDialogCls .itemCls .el-icon::before{
		font-size: 12px;
	}
	.addWhiteBlackDialogCls .cellIdMessage-default{
		margin-left: 18px;
		color: #999999;
	}
	.addWhiteBlackDialogCls .cellIdMessage-error{
		margin-left: 18px;
		color: #FA5555!important;
	}
	#egwBlackAndWhiteList .selectBlukBoxCls .selectTableBoxCls {
		height: 300px;
	}
</style>
<div class="pageDefault" id='egwBlackAndWhiteList' style="position:relative;overflow:hidden;border:none;">
	<div class="container">
		<div class="blackAndWhiteListMainBox">
			<!-- 操作按钮 -->
			<div class="operations">
				<div class="newIconBoxCls-bt" placeholder="<%=rb.getString("GuanBi")%>">
					<span class="el-icon el-icon-circle-close" @click="closeListPage"></span>
				</div>
			</div>
			<div class="list_left_box">
				<el-ctable
					:url="DeviceListTableUrl"
					:query-params="queryDeviceListParams" 
					ref="DeviceTableList" 
					id="DeviceTableList"
					:height="height" 
					highlight-current-row="true"
					:row-key="'serial_number'"
					:page-size="pageSize" 
					:page-list="pageList" 
					pagination="true">
						<!-- 列表toolbar -->
					<template slot="toolbar">
						<div class="headQueryBox">
							<h3 style="padding-left: 10px;"><%=rb.getString("SheBeiLieBiaoBiaoTi")%></h3>
							<el-query type="normal" @query="queryDeviceListTable" placeholder="<%=rb.getString("eGWBianMa")%> / <%=rb.getString("EGWIP")%>"></el-query>
						</div>
					</template>
						<!-- 列表columns -->
					<el-table-column prop="serial_number" label="<%=rb.getString("eGWSheBeiBianMa")%>" min-width="150"></el-table-column>
					<el-table-column prop="gw_ip" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
					<el-table-column prop="gw_port" label="<%=rb.getString("EGWDuanKou")%>" min-width="100"></el-table-column>
					<el-table-column prop="generation" label="" min-width="140">
						<template slot-scope="scope">
							<div style="display:flex;align-items: center;">
								<span v-if="formatterDeviceType(scope.row.generation) == '4G' || formatterDeviceType(scope.row.generation) == '4/5G'" style="color: #4D84FF;cursor: pointer;margin-right:10px;" @click="blockAndWhiteListBtnClick('4G',scope.row)">4G黑白名单</span>
								<span v-if="formatterDeviceType(scope.row.generation) == '5G' || formatterDeviceType(scope.row.generation) == '4/5G'" style="color: #4D84FF;cursor: pointer;"  @click="blockAndWhiteListBtnClick('5G',scope.row)">5G黑白名单</span>
							</div>
						</template>
					</el-table-column>
				</el-ctable>
			</div>
			<div class="list_right_box" v-show="blockAndWhiteListType">
				<div class="right_box_header">
					<span style="margin-left:20px;color:#363B4E;font-weight: bold;">SN:{{rowDeviceData.serial_number}}</span>
					<span class="modeLabelCls">Mode</span>
					<el-select v-model='currentMode' size="mini" @change="currentModeChange" :disabled="!rowDeviceData || !rowDeviceData.serial_number || !egwOptBtnShow">
						<el-option label='Allow' value='White'></el-option>
						<el-option label='Block' value='Black'></el-option>
						<el-option label="Disabled" value="None"></el-option>
					</el-select>
					<span class="promptInfoCls" v-if="currentMode == 'White'"><%=rb.getString("BaiMingDanMoShiTiShi")%></span>
					<span class="promptInfoCls" v-if="currentMode == 'Black'"><%=rb.getString("HeiMingDanMoShiTiShi")%></span>
					<span class="promptInfoCls" v-if="currentMode == 'None'"><%=rb.getString("JinYongMoShiTiShi")%></span>
				</div>
				<div class="whiteListTable">
					<div class="operations">
						<div class="newIconBoxCls-bt" tip="<%=rb.getString("TianJia")%>">
							<span class="el-icon el-icon-circle-add" @click="addList('white')"></span>
						</div>
					</div>
					<el-ctable
						:url="whiteListTableUrl"
						:query-params="queryWhiteParams" 
						ref="whiteTableList" 
						id="whiteTableList"
						:time="6"
						:height="height" 
						@selection-change='selectWhite'
						:page-size="pageSize" 
						:page-list="pageList" 
						pagination="true">
							<!-- 列表toolbar -->
						<template slot="toolbar">
							<div class="toolbarHeadBtnBoxCls">
								<div v-show="egwOptBtnShow" class="selectBlukBoxCls">
									<div class="selectMain">
										<div class="bulkSelectBtnBoxCls"  @click="openBulkSelectWhiteTable">
											<span class="el-icon-selected el-icon"></span>
											<span class="bulkSelectNumBoxCls">( {{selectWhiteList.length}} )</span>
										</div>
										<div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectWhiteShow">
											<div class="selectBoxTitle">
												<span><%=rb.getString("YiXuan")%></span>
												<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectWhiteTable"></span>
											</div>
											<div class="selectBoxMain">
												<div class="tableInfoCls">
													<div class="tableInfoHeader">
														<div>{{placeholderText}}</div>
														<div @click="clearWhiteBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
													</div>
													<el-ctable 
														id="bulkSelectWhiteListTable" 
														ref="bulkSelectWhiteListTable" 
														:data="selectWhiteList" 
														:showHeader="false"
														:rownumber="false"
														:front-pagination="true"
														height="200px" pagination="true" >
														<el-table-column prop="id" v-if="false"></el-table-column>
														<el-table-column width="588">
															<template slot-scope="scope" >
																<div class="tableItemCls">
																	<span v-if="blockAndWhiteListType == '4G'">{{scope.row.eci}}</span>
																	<span v-if="blockAndWhiteListType == '5G'">{{scope.row.gnb_id}}</span>
																	<span @click="delWhiteBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
																</div>
															</template>
														</el-table-column>
													</el-ctable>
												</div>
											</div>
										</div>
									</div>
								</div>
								<div v-show="egwOptBtnShow" :class="selectWhiteList.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="deleteList('all','all','white')">
									<span class="el-icon el-icon-operation-delete"></span>
									<span><%=rb.getString("ShanChu")%></span>
								</div>
							</div>
							<div style="display: flex;align-items: center;padding-top:10px;">
								<h3 style="padding-left: 10px;"><%=rb.getString("BaiMingDan")%></h3>
								<el-query type="normal" @query="queryWhiteTable" :placeholder="placeholderText"></el-query>
							</div>
							
						</template>
							<!-- 列表columns -->
						<el-table-column v-if="egwOptBtnShow" type="selection" width="45"></el-table-column>
						<el-table-column prop="" width="80">
							<template slot-scope="scope">
								<div style="display:flex;align-items: center;">
									<el-tooltip content='<%=rb.getString("ShanChu")%>' placement='bottom'>
										<span class="el-icon el-icon-operation-delete" @click="deleteList('single',scope.row,'white')"></span>
									</el-tooltip>
									<el-tooltip content='<%=rb.getString("XiuGai")%>' placement='bottom'>
										<span class="el-icon el-icon-operation-edit" style="margin-left:5px;" @click="editList('white',scope.row)"></span>
									</el-tooltip>
								</div>
							</template>
						</el-table-column>
						<el-table-column v-if="blockAndWhiteListType == '4G'" key="eci" label="ECI" min-width="200" prop="eci" show-overflow-tooltip></el-table-column>
						<el-table-column v-if="blockAndWhiteListType == '4G'" key="enb_id" label="eNodeB ID" min-width="200" prop="enb_id" show-overflow-tooltip></el-table-column>
						<el-table-column v-if="blockAndWhiteListType == '4G'" key="cell_list" label="Cell ID" min-width="200" prop="cell_list" show-overflow-tooltip></el-table-column>
						<el-table-column v-if="blockAndWhiteListType == '5G'" key="gnb_id" label="<%=rb.getString("GnodebId")%>" prop="gnb_id"  min-width="150" show-overflow-tooltip></el-table-column>
						<el-table-column key="operation_time" label="<%=rb.getString("CaoZuoShiJian")%>" prop="operation_time"  min-width="150"></el-table-column>
					</el-ctable>
				</div>
				<div class="blackListTable">
					<div class="operations">
						<div class="newIconBoxCls-bt" tip="<%=rb.getString("TianJia")%>">
							<span class="el-icon el-icon-circle-add" @click="addList('black')"></span>
						</div>
					</div>
					<el-ctable
						:url="blackListTableUrl"
						:query-params="queryBlackParams" 
						ref="blackTableList" 
						id="blackTableList"
						:time="6"
						:height="height" 
						@selection-change='selectBlack'
						:page-size="pageSize" 
						:page-list="pageList" 
						pagination="true">
							<!-- 列表toolbar -->
						<template slot="toolbar">
							<div class="toolbarHeadBtnBoxCls">
								<div v-show="egwOptBtnShow" class="selectBlukBoxCls">
									<div class="selectMain">
										<div class="bulkSelectBtnBoxCls"  @click="openBulkSelectBlackTable">
											<span class="el-icon-selected el-icon"></span>
											<span class="bulkSelectNumBoxCls">( {{selectBlackList.length}} )</span>
										</div>
										<div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectBlackShow">
											<div class="selectBoxTitle">
												<span><%=rb.getString("YiXuan")%></span>
												<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectBlackTable"></span>
											</div>
											<div class="selectBoxMain">
												<div class="tableInfoCls">
													<div class="tableInfoHeader">
														<div>{{placeholderText}}</div>
														<div @click="clearBlackBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
													</div>
													<el-ctable 
														id="bulkSelectBlackListTable" 
														ref="bulkSelectBlackListTable" 
														:data="selectBlackList" 
														:showHeader="false"
														:rownumber="false"
														:front-pagination="true"
														height="200px" pagination="true" >
														<el-table-column prop="id" v-if="false"></el-table-column>
														<el-table-column width="588">
															<template slot-scope="scope" >
																<div class="tableItemCls">
																	<span v-if="blockAndWhiteListType == '4G'">{{scope.row.eci}}</span>
																	<span v-if="blockAndWhiteListType == '5G'">{{scope.row.gnb_id}}</span>
																	<span @click="delBlackBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
																</div>
															</template>
														</el-table-column>
													</el-ctable>
												</div>
											</div>
										</div>
									</div>
								</div>
								<div v-show="egwOptBtnShow" :class="selectBlackList.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="deleteList('all','all','black')">
									<span class="el-icon el-icon-operation-delete"></span>
									<span><%=rb.getString("ShanChu")%></span>
								</div>
							</div>
							<div style="display: flex;align-items: center;padding-top:10px;">
								<h3 style="padding-left: 10px;"><%=rb.getString("HeiMingDan")%></h3>
								<el-query type="normal" @query="queryBlackTable"  :placeholder="placeholderText"></el-query>
							</div>
						</template>
							<!-- 列表columns -->
						<el-table-column v-if="egwOptBtnShow" type="selection" width="45"></el-table-column>
						<el-table-column prop="" width="80">
							<template slot-scope="scope">
								<div style="display:flex;align-items: center;">
									<el-tooltip content='<%=rb.getString("ShanChu")%>' placement='bottom'>
										<span class="el-icon el-icon-operation-delete" @click="deleteList('single',scope.row,'black')"></span>
									</el-tooltip>
									<el-tooltip content='<%=rb.getString("XiuGai")%>' placement='bottom'>
										<span class="el-icon el-icon-operation-edit" style="margin-left:5px;" @click="editList('black',scope.row)"></span>
									</el-tooltip>
								</div>
							</template>
						</el-table-column>
						<el-table-column v-if="blockAndWhiteListType == '4G'" key="ECI" label="ECI" min-width="200" prop="eci" show-overflow-tooltip></el-table-column>
						<el-table-column v-if="blockAndWhiteListType == '4G'" key="enb_id" label="eNodeB ID" min-width="200" prop="enb_id" show-overflow-tooltip></el-table-column>
						<el-table-column v-if="blockAndWhiteListType == '4G'" key="cell_list" label="Cell ID" min-width="200" prop="cell_list" show-overflow-tooltip></el-table-column>
						<el-table-column v-if="blockAndWhiteListType == '5G'" key="gnb_id" label="<%=rb.getString("GnodebId")%>" prop="gnb_id"  min-width="150" show-overflow-tooltip></el-table-column>
						<el-table-column key="operation_time" label="<%=rb.getString("CaoZuoShiJian")%>" prop="operation_time" min-width="150"></el-table-column>
					</el-ctable>
				</div>
			</div>
		</div>
    </div>
	<!--新增4G黑白名单弹窗-->
	<el-dialog :title='dialogTitle' class="addWhiteBlackDialogCls" :visible.sync="showWhiteBlackDialog" top="30vh" ref="addWhiteBlackDialog" 
		:width="dialogWidth" :close-on-click-modal="false"  @close='closeDialog' append-to-body>
		<el-form  :model="addListForm" ref="addListForm" :rules="addListRules" label-position="left" id="addListForm">
			<el-form-item  style="margin-left:40px;" label="eNodeB ID" prop='enb_id' label-width="100px" class='validate-item'>
				<el-input v-model.trim='addListForm.enb_id' :disabled="optType == 'edit'"  style="width:200px;padding-top:5px;">
					<template slot="append">{{addListForm.enodebIdErr}}</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='cell_id' :class="cellInputErrorClass" style="margin-left:40px;margin-bottom:0px;position:relative;" label="Cell ID" label-width="100px">
				<el-input v-model.trim='addListForm.cell_id' style="width:150px;padding-top:5px;"></el-input>
				<span v-if="addListForm.cellList.length < 20" @click="addCellId" class='el-icon el-icon-plus' style='position:absolute;left:130px;top:10px;'></span>
				<span v-else class='el-icon el-icon-plus disabled' style='position:absolute;left:130px;top:10px;'></span>
				<span :class="cellIdMessageClass">{{addListForm.message}}</span>
			</el-form-item>
			<div class="itemListBoxCls">
				<div v-for="(item,index) in addListForm.cellList" class="itemCls">
					<span style="font-size:12px;">{{item}}</span>
					<span class="el-icon el-icon-close" style="margin-left:5px;" @click="cellListDel(index,item)"></span>
				</div>
			</div>
		</el-form>
		<span slot="footer">
			<div>
				<el-button type="primary" @click="addListSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeDialog"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</span>
	</el-dialog>
	<!--新增5G黑白名单弹窗-->
	<el-dialog :title='dialogTitle' class="addWhiteBlackDialogCls" :visible.sync="show5GWhiteBlackDialog" top="30vh" ref="add5GWhiteBlackDialog" 
		:width="dialogWidth" :close-on-click-modal="false"  @close='close5GWhiteBlackDialog' append-to-body>
		<el-form  :model="add5GWhiteBlackListForm" ref="add5GWhiteBlackListForm" :rules="add5GWhiteBlackListRules" label-position="left" id="add5GWhiteBlackListForm">
			
			<el-form-item prop='cell_id' :class="cellInputErrorClass" style="margin-left:40px;margin-bottom:0px;position:relative;" label="<%=rb.getString("GnodebId")%>" label-width="100px">
				<el-input v-model.trim='add5GWhiteBlackListForm.gnb_id' style="width:150px;padding-top:5px;"></el-input>
				<span v-if="add5GWhiteBlackListForm.gnbIdList.length < 20" @click="addGnbId" class='el-icon el-icon-plus' style='position:absolute;left:130px;top:10px;'></span>
				<span v-else class='el-icon el-icon-plus disabled' style='position:absolute;left:130px;top:10px;'></span>
				<span :class="cellIdMessageClass">{{add5GWhiteBlackListForm.message}}</span>
			</el-form-item>
			<div class="itemListBoxCls">
				<div v-for="(item,index) in add5GWhiteBlackListForm.gnbIdList" class="itemCls">
					<span style="font-size:12px;">{{item}}</span>
					<span class="el-icon el-icon-close" style="margin-left:5px;" @click="gnbIdListDel(index,item)"></span>
				</div>
			</div>
			<el-form-item>
				<el-input style="display: none;"></el-input>
			</el-form-item>
		</el-form>
		<span slot="footer">
			<div>
				<el-button type="primary" @click="add5GWhiteBlackListSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="close5GWhiteBlackDialog"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</span>
	</el-dialog>
</div>
<script type="text/javascript">
var currentModeTimer;
var egwBlackAndWhiteListVue = new Vue({
	el:'#egwBlackAndWhiteList',
	data(){
		var vm = this;
		var validateRange = (rule,value,callback)=>{
			var min = rule.min;
			var max = rule.max;
			var mag = rule.mag;
			var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
			if(value == '' || value == undefined || value == null){
				vm.addListForm.enodebIdErr = '<%=rb.getString("FanWei")%>：0~1048575'
				callback(new Error(mag))
			}else if(value.trim() == vm.defaultEnodebId){
				vm.addListForm.enodebIdErr = '<%=rb.getString("FanWei")%>：0~1048575'
				callback();
			}else{
				if(reg.test(value) && value >= min && value <= max){
					axios.post("${ctx}/egw/enbWhiteBlackList/checkECIIsExist.action",stringify({
						egw_serial_number:vm.rowDeviceData.serial_number,
						enb_id:vm.addListForm.enb_id,
						switchFalg:vm.listType
					})).then(function(response){
						var data = response.data;
						if(data){
							vm.addListForm.enodebIdErr = '<%=rb.getString("YiCunZai")%>'
							callback(new Error(mag))
						}else{
							vm.addListForm.enodebIdErr = '<%=rb.getString("FanWei")%>：0~1048575'
							callback()
						}
					}).catch(function(error){
						vm.addListForm.enodebIdErr = '<%=rb.getString("FanWei")%>：0~1048575'
						callback()
					})
				}else{
					vm.addListForm.enodebIdErr = '<%=rb.getString("FanWei")%>：0~1048575'
					callback(new Error(mag))
				}
			}
			
		};
		return {
			DeviceListTableUrl:'${ctx}/egw/enbWhiteBlackList/getEGWDeviceList.action',
			blackListTableUrl:"",
			whiteListTableUrl:"",
			selectBlackList:[],
			selectWhiteList:[],
			queryDeviceListParams:{
				timeZone:timeZone,
				searchText:'',
			},
			queryBlackParams:{
				timeZone:timeZone,
				searchText:'',
				egw_serial_number:'',
			},
			queryWhiteParams:{
				timeZone:timeZone,
				searchText:'',
				egw_serial_number:'',
			},
            height:'100%',
            pageSize:100,
			pageList:[50,100,200,500],
			rowDeviceData:{
				serial_number:'',
			},
			currentMode:'White',
			firstLoad:true,
			dialogTitle:'<%=rb.getString("TianJia")%>',
			dialogWidth:'600px',
			showWhiteBlackDialog:false,
			addListForm:{
				enb_id:'',
				cell_id:'',
				cellList:[],
				enodebIdErr:'<%=rb.getString("FanWei")%>：0~1048575',
				message:'<%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%>：0~255'
			},
			addListRules:{
				enb_id:[
					{validator:validateRange,min:0,max:1048575,mag:'<%=rb.getString("FanWei")%>：0~1048575'}
				],
			},
			show5GWhiteBlackDialog:false,
			add5GWhiteBlackListForm:{
				gnb_id:'',
				gnbIdList:[],
				message:'<%=rb.getString("FanWei")%>：0~4294967295,<%=rb.getString("ZhengXing")%>'
			},
			add5GWhiteBlackListRules:{},
			listType:'', // 黑白名单 类型 white  black
			cellIdErrorShow:false,
			defaultEnodebId:'',
			optType:'',
			blockAndWhiteListType:'',
			bulkSelectWhiteShow:false,
			bulkSelectBlackShow:false,
		}
	},
    computed:{
		isSuperAdmin() {
			return is_super_user == 'true';
		},
		// 图表样式
		cellIdMessageClass(){
			return {
				'cellIdMessage-default': true,
				'cellIdMessage-error': this.cellIdErrorShow
			};
		},
		cellInputErrorClass(){
			return {
				'is-error': this.cellIdErrorShow,
			};
		},
		egwOptBtnShow() {
			return writableMap['CODE_EGW'] == true;
		},
		placeholderText(){
			return this.blockAndWhiteListType == '4G' ? 'ECI' : '<%=rb.getString("GnodebId")%>';
		}
    },
	watch:{},
	methods:{
		//关闭黑白名单页面
		closeListPage(){
			var vm = this;
			if(vm.blockAndWhiteListType){
				vm.blockAndWhiteListType = '';
			}else{
				accessVue.$refs.egwListSlide.hide();
			}
		},
		// 黑名单选择事件
		selectBlack(selection){
			var vm = this;
			if(vm.selectWhiteList.length>0){
				vm.$refs.whiteTableList.clearSelection();
			}
			vm.selectBlackList = selection
		},
		// 点击4G 5G黑白名单按钮
		blockAndWhiteListBtnClick(type,row){
			var vm = this;
			vm.blockAndWhiteListType = type;
			vm.rowDeviceData = row;
			vm.getInitCurrentMode();
			clearInterval(currentModeTimer);
			currentModeTimer = setInterval(function(){
				var egwBlackAndWhite = $("#egwBlackAndWhiteList");			
				if(!egwBlackAndWhite.length || !vm.blockAndWhiteListType) {
					clearInterval(currentModeTimer);
					return;
				}
				vm.getInitCurrentMode();
			},6000);
			if(vm.blockAndWhiteListType == '4G'){
				vm.blackListTableUrl = "${ctx}/egw/enbWhiteBlackList/getBlackList.action";
				vm.whiteListTableUrl = "${ctx}/egw/enbWhiteBlackList/getWhiteList.action";
			}else{
				vm.blackListTableUrl = "${ctx}/egw/gnbWhiteBlackList/getBlackList.action";
				vm.whiteListTableUrl = "${ctx}/egw/gnbWhiteBlackList/getWhiteList.action";
			}
			vm.queryWhiteParams.egw_serial_number = row.serial_number;
			vm.queryBlackParams.egw_serial_number = row.serial_number;
		},
		getInitCurrentMode(){
			var vm = this,
				url = '',
				params = {
					egw_serial_number:vm.rowDeviceData.serial_number,
				};
			if(vm.blockAndWhiteListType == '4G'){
				url = "${ctx}/egw/enbWhiteBlackList/getSwitchOption.action";
			}else{
				url = "${ctx}/egw/gnbWhiteBlackList/getSwitchOption.action";
			}
			axios.post(url,stringify(params)).then(function(response){
				var data = response.data;
				vm.currentMode = data;
			})
		},
		// 当前模式改变
		currentModeChange(val){
			var vm = this,
				url = '',
				params = {
					egw_serial_number:vm.rowDeviceData.serial_number,
					switchOption:val
				},
				codeList={
					'White':{header:'<%=rb.getString("QueRenQieHuanWeiBaiMingDanMoShi")%>',footer:'<%=rb.getString("BaiMingDanMoShiTiShi")%>'},
					'Black':{header:'<%=rb.getString("QueRenQieHuanWeiHeiMingDanMoShi")%>',footer:'<%=rb.getString("HeiMingDanMoShiTiShi")%>'},
					'None':{header:'<%=rb.getString("QueRenQieHuanWeiJinYongMoShi")%>',footer:'<%=rb.getString("JinYongMoShiTiShi")%>'},
				};
			if(vm.blockAndWhiteListType == '4G'){
				url = "${ctx}/egw/enbWhiteBlackList/updateSwitchOption.action";
			}else{
				url = "${ctx}/egw/gnbWhiteBlackList/updateSwitchOption.action";
			}
			var str ='<div style="font-size:14px;color:#333333">'+ codeList[val].header +'</div>'+'<div style="font-size:12px;color:#999999">'+ codeList[val].footer +'</div>';
			vm.$confirm(str,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					dangerouslyUseHTMLString:true
				}).then(function(){
					axios.post(url,stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
								message:'<%=rb.getString("ChengGong")%>',
								type:'success',
							})
							vm.getInitCurrentMode();
						}else{
							vm.getInitCurrentMode();
							vm.$message.error(data["message"])
						}
					})
				}).catch(function(){
					vm.getInitCurrentMode();
				})
			
		},
		// 白名单选择
		selectWhite(selection){
			var vm = this;
			if(vm.selectBlackList.length>0){
				vm.$refs.blackTableList.clearSelection();
			}
			vm.selectWhiteList = selection
		},
		// 设备列表 模糊查询
		queryDeviceListTable(val){
			var vm = this;
			vm.queryDeviceListParams.searchText= val;
		},
		// 黑名单 模糊查询
		queryBlackTable(val){
			var vm = this;
			vm.queryBlackParams.searchText= val;
		},
		// 白名单 模糊查询
		queryWhiteTable(val){
			var vm = this;
			vm.queryWhiteParams.searchText= val;
		},
		// 删除
		deleteList(singleType,row,listType){
			var vm = this,
				url = "",
				target,
				params = {
					index:row.index,
					egw_serial_number:vm.rowDeviceData.serial_number
				},
				dataList = [];

			if(listType == "black"){
				if(vm.blockAndWhiteListType == '4G'){
					url = "${ctx}/egw/enbWhiteBlackList/delBlackList.action";
				}else{
					url = "${ctx}/egw/gnbWhiteBlackList/delBlackList.action";
				}
				target = vm.$refs.blackTableList;
				if(singleType == "single"){
					params.index = row.index;
				}else{
					vm.selectBlackList.map((item)=>{
						dataList.push(item.index)
					})
					params.index = dataList.join(',')
				}
			}else{
				if(vm.blockAndWhiteListType == '4G'){
					url = "${ctx}/egw/enbWhiteBlackList/delWhiteList.action";
				}else{
					url = "${ctx}/egw/gnbWhiteBlackList/delWhiteList.action";
				}
				target = vm.$refs.whiteTableList;
				if(singleType == "single"){
					params.index = row.index;
				}else{
					vm.selectWhiteList.map((item)=>{
						dataList.push(item.index)
					})
					params.index = dataList.join(',')
				}
			}
			vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
				customClass:'warningConfirm',
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				closeOnClickModal:false
			}).then(() => {
				axios.post(url,stringify(params)).then(function(response){
					var data = response.data;
					var message = '<%=rb.getString("ChengGong")%>';
					if(data["success"]){
						vm.$message({
							message:message,
							type:'success',
						})
						target.refresh();
					}else{
						vm.$message.error(data["message"])
					}
				})
			}).catch(() => {
				
			})
		},
		/**
		*  打开新增弹窗
		* @param type{string}  1.white 2. black
		*/ 
		addList(listType){
			var vm = this;
			vm.listType = listType;
			vm.optType = 'add';
			vm.dialogTitle = '<%=rb.getString("TianJia")%>';
			if(vm.blockAndWhiteListType == '4G'){
				vm.showWhiteBlackDialog = true;
			}else{
				vm.show5GWhiteBlackDialog = true;
			}
		},
		// 打开修改弹窗
		editList(listType,row){
			var vm = this;
			vm.listType = listType;
			vm.optType = 'edit';
			vm.dialogTitle = '<%=rb.getString("XiuGai")%>';
			if(vm.blockAndWhiteListType == '4G'){
				Object.assign(vm.addListForm,row);
				vm.defaultEnodebId = row.enb_id;
				vm.addListForm.cellList = row.cell_list.split(',');
				vm.showWhiteBlackDialog = true;
			}else{
				Object.assign(vm.add5GWhiteBlackListForm,row);
				vm.add5GWhiteBlackListForm.gnbIdList = row.gnb_id.split(',');
				vm.show5GWhiteBlackDialog = true;
			}
		},
		addCellId(){
			var vm = this;
				value = vm.addListForm.cell_id,
				reg = /^\d+(?:-\d+)*$/,
				isAdd = true;
			if(value == '' || value == undefined || value == null){
				vm.cellIdErrorShow = false;
				return
			}else{
				if(reg.test(value)){
					var arr = value.split('-');
					if(arr.length > 2 ){
						vm.cellIdErrorShow = true;
						return
					}else{
						if(arr.length < 2){
							if(parseFloat(value)<= 255){
								vm.addListForm.cellList.map((item)=>{
									var itemList = item.split('-');
									if(itemList.length < 2){
										if(value == item){
											isAdd = false;
											return
										}
									}else{
										if(vm.isRangeIn(value,itemList[1],itemList[0])){
											isAdd = false;
											return
										} 
									}
								})
								if(isAdd){
									
								}else{
									vm.cellIdErrorShow = true;
									return
								}
							}else{
								vm.cellIdErrorShow = true;
								return
							}
						}else{
							if(vm.isCorrectRange(value)){
								vm.addListForm.cellList.map((item)=>{
									var itemList = item.split('-');
									var valueList = value.split('-');
									if(itemList.length < 2){
										if(vm.isRangeIn(item,valueList[1],valueList[0])){
											isAdd = false;
											return
										} 
									}else{
										if(vm.isIntersection(item,value)){
											isAdd = false;
											return
										} 
									}
								})
								if(isAdd){
								
								}else{
									vm.cellIdErrorShow = true;
									return
								}
							}else{
								vm.cellIdErrorShow = true;
								return
							}
						}
					}
				}else{
					vm.cellIdErrorShow = true;
					return
				}
			}
			vm.cellIdErrorShow = false;
			vm.addListForm.cellList.push(vm.addListForm.cell_id);
			vm.addListForm.cell_id = '';
			vm.addListForm.message = '<%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%>：0~255';
			
		},
		// 新增弹窗 cell删除
		cellListDel(index,item){
			var vm = this;
			vm.addListForm.cellList.splice(index,1);
		},
		// 黑白名单新增提交
		addListSubmit(){
			var vm = this,
				url = '',
				message = '',
				params = {};

			vm.$refs.addListForm.validate((valid) => {
	    		if(valid){
					if(vm.addListForm.cellList.length<1){
						vm.cellIdErrorShow = true;
						vm.addListForm.message = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
						return
					}
					params.egw_serial_number = vm.rowDeviceData.serial_number;
	    			params.enb_id = vm.addListForm.enb_id;
	    			params.cell_list = vm.addListForm.cellList.join(',');
					if(vm.optType == 'add'){
						if(vm.listType == 'white'){
							url = "${ctx}/egw/enbWhiteBlackList/addWhiteList.action";
						}else{
							url = "${ctx}/egw/enbWhiteBlackList/addBlackList.action";
						}
					}else{
						params.eci = vm.addListForm.eci;
						params.index = vm.addListForm.index;
						if(vm.listType == 'white'){
							url = "${ctx}/egw/enbWhiteBlackList/updateWhiteList.action";
						}else{
							url = "${ctx}/egw/enbWhiteBlackList/updateBlackList.action";
						}
					}
					axios.post(url,stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
								message:'<%=rb.getString("ChengGong")%>',
								type:'success',
							})
							vm.$refs.whiteTableList.refresh();
							vm.$refs.blackTableList.refresh();
							vm.showWhiteBlackDialog = false;
							vm.closeDialog();
						}else{
							vm.$message.error(data["message"])
						}
					})
	    		}else{
	    			return false;
	    		}
	    	})
		},
		// 新增弹窗关闭
		closeDialog(){
			var vm = this,
				params = {
					enb_id:'',
					cell_id:'',
					cellList:[],
					message:'<%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%>：0~255'
				};
			vm.defaultEnodebId = '';
			vm.cellIdErrorShow = false;
			vm.showWhiteBlackDialog = false;
			Object.assign(vm.addListForm,params)
		},
		// 5G 黑白名单添加gnb id
		addGnbId(){
			var vm = this;
				value = vm.add5GWhiteBlackListForm.gnb_id,
				reg = /^\d+$/,
				isAdd = true;
			if(value == '' || value == undefined || value == null){
				vm.cellIdErrorShow = false;
				return
			}else{
				if(reg.test(value) && value >= 0 && value <= 4294967295){
					
				}else{
					vm.cellIdErrorShow = true;
					return
				}
			}
			vm.cellIdErrorShow = false;
			vm.add5GWhiteBlackListForm.gnbIdList.push(vm.add5GWhiteBlackListForm.gnb_id);
			vm.add5GWhiteBlackListForm.gnb_id = '';
			vm.addListForm.message = '<%=rb.getString("FanWei")%>：0~4294967295,<%=rb.getString("ZhengXing")%>';
			
		},
		// 5G 黑白名单新增弹窗 gnb id删除
		gnbIdListDel(index,item){
			var vm = this;
			vm.add5GWhiteBlackListForm.gnbIdList.splice(index,1);
		},
		// 5G黑白名单新增提交
		add5GWhiteBlackListSubmit(){
			var vm = this,
				url = '',
				message = '',
				params = {};

			vm.$refs.add5GWhiteBlackListForm.validate((valid) => {
	    		if(valid){
					if(vm.add5GWhiteBlackListForm.gnbIdList.length<1){
						vm.cellIdErrorShow = true;
						vm.add5GWhiteBlackListForm.message = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
						return
					}
					params.egw_serial_number = vm.rowDeviceData.serial_number;
	    			params.gnb_id = vm.add5GWhiteBlackListForm.gnbIdList.join(',');
					if(vm.optType == 'add'){
						if(vm.listType == 'white'){
							url = "${ctx}/egw/gnbWhiteBlackList/addWhiteList.action";
						}else{
							url = "${ctx}/egw/gnbWhiteBlackList/addBlackList.action";
						}
					}else{
						params.index = vm.add5GWhiteBlackListForm.index;
						if(vm.listType == 'white'){
							url = "${ctx}/egw/gnbWhiteBlackList/updateWhiteList.action";
						}else{
							url = "${ctx}/egw/gnbWhiteBlackList/updateBlackList.action";
						}
					}
					axios.post(url,stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
								message:'<%=rb.getString("ChengGong")%>',
								type:'success',
							})
							vm.$refs.whiteTableList.refresh();
							vm.$refs.blackTableList.refresh();
							vm.show5GWhiteBlackDialog = false;
							vm.closeDialog();
						}else{
							vm.$message.error(data["message"])
						}
					})
	    		}else{
	    			return false;
	    		}
	    	})
		},
		// 新增弹窗关闭
		close5GWhiteBlackDialog(){
			var vm = this,
				params = {
					gnb_id:'',
					gnbIdList:[],
					message:'<%=rb.getString("FanWei")%>：0~4294967295,<%=rb.getString("ZhengXing")%>'
				};
			vm.cellIdErrorShow = false;
			vm.show5GWhiteBlackDialog = false;
			Object.assign(vm.add5GWhiteBlackListForm,params)
		},
		// 判断一个数是否在区间内
		isRangeIn(val,maxNum,minNum){
			var num = parseFloat(val),
				max = parseFloat(maxNum),
				min = parseFloat(minNum);
			if(num <= max && num >= min){
				return true
			}
			return false
		},
		// 是否符合例如：1-10 正确范围值
		isCorrectRange(val){
			var min = parseFloat(val.split('-')[0]),
				max = parseFloat(val.split('-')[1]);
			if(min >= max ){
				return false
			}
			if(max > 255){
				return false
			}
			return true
		},
		// 判断两个区间是否有交集 例：range1:1-10  range2:9-11
		isIntersection(range1,range2){
			var max = [parseFloat(range1.split('-')[0]),parseFloat(range2.split('-')[0])],
				min = [parseFloat(range1.split('-')[1]),parseFloat(range2.split('-')[1])];
			if(Math.max.apply(null,max) <= Math.min.apply(null,min)){
				return true
			}
			return false
		},
		
		// 打开白名单已选弹窗
		openBulkSelectWhiteTable(){
			var vm = this;
			vm.bulkSelectWhiteShow = true
		},
		// 关闭白名单已选弹窗
		closeBulkSelectWhiteTable(){
			var vm = this;
			vm.bulkSelectWhiteShow = false;
		},
		// 白名单设备已选表格 清空事件
		clearWhiteBulkSelected(){
			var vm = this;

			vm.$refs["whiteTableList"].clearSelection();
		},
		// 白名单设备已选表格 单个删除事件
		delWhiteBulkSelected(rows){
			var vm = this,
				tabs = 'whiteTableList',
				rowKey = vm.blockAndWhiteListType == '4G' ? 'eci' : 'gnb_id';
			vm.selectWhiteList = vm.selectWhiteList.filter((items)=>{
				return items[rowKey] != rows[rowKey]
			});
			var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
				irow= selection.filter((items)=>{
					return items[rowKey] == rows[rowKey]
				})[0];
			vm.$refs[tabs].toggleRowSelection(irow,false);
			var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
			vm.$refs[tabs].ckList.splice(idx,1);
		},
		// 打开黑名单已选弹窗
		openBulkSelectBlackTable(){
			var vm = this;
			vm.bulkSelectBlackShow = true
		},
		// 关闭黑名单已选弹窗
		closeBulkSelectBlackTable(){
			var vm = this;
			vm.bulkSelectBlackShow = false;
		},
		// 黑名单设备已选表格 清空事件
		clearBlackBulkSelected(){
			var vm = this;

			vm.$refs["blackTableList"].clearSelection();
		},
		// 黑名单设备已选表格 单个删除事件
		delBlackBulkSelected(rows){
			var vm = this,
				tabs = 'blackTableList',
				rowKey = vm.blockAndWhiteListType == '4G' ? 'eci' : 'gnb_id';
			vm.selectBlackList = vm.selectBlackList.filter((items)=>{
				return items[rowKey] != rows[rowKey]
			});
			var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
				irow= selection.filter((items)=>{
					return items[rowKey] == rows[rowKey]
				})[0];
			vm.$refs[tabs].toggleRowSelection(irow,false);
			var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
			vm.$refs[tabs].ckList.splice(idx,1);
		},
		formatterDeviceType(generation){
			var vm = this,
				types = '4G';

			if(generation == 'SigGW4G' ||  generation == 'SigGW4G+SeGW'){
				types = '4G';
			}else if(generation == 'SigGW5G' || generation == 'SigGW5G+SeGW'){
				types = '5G';
			}else if(generation == 'SigGW4/5G' || generation == 'SigGW4/5G+SeGW'){
				types = '4/5G';
			}
			return types
		},
	},
	mounted(){}
	
})

</script> 
