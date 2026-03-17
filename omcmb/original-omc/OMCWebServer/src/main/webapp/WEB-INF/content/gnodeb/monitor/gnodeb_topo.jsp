<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style type="text/css">
	#gNBTopoConfig .h100{height:100%}
	#gNBTopoConfig .el-tabs__content {padding-top: 14px;}
	
	#gNBTopoConfig .operationTip {cursor: pointer;}
	#gNBTopoConfig .euStatus {font-size: 22px;}
	#gNBTopoConfig .statusTip {vertical-align: top;margin-left: 5px;}
	#gNBTopoConfig .ruWarp {flex: 1;display: flex;flex-direction: column;overflow: auto}
	#gNBTopoConfig .closePage{position:absolute; top: 6px; font-size:30px; z-index:999;}
	#gNBTopoConfig .topoContent{width: 100%;height: 100%; -webkit-tap-highlight-color:rgba(255,0,0,0)}
	#gNBTopoConfig .el-collapse-item{ position: relative;}
	#gNBTopoConfig .el-collapse-item__arrow{
		position:absolute;
		left:50px;
		top:-1px;
	}
	#gNBTopoConfig .el-icon-arrow-right{
		font-size:16px;
	}
	#gNBTopoConfig .el-icon-arrow-right:before{
		content:"\e639";
		color:#333333;
	}
	#gNBTopoConfig .is-active.el-icon-arrow-right:before{
		content:"\e638";
		color:#333333;
	}
	#gNBTopoConfig .el-collapse-item__arrow.is-active{
		transform:rotate(0deg);
	}
	#gNBTopoConfig .titleWarp {
		display: flex;
		margin-bottom: 22px;
		margin-left: 12px;
	}
	#gNBTopoConfig .titleWarp span{
		font-size: 14px;
		color: #333333;
		font-weight: bold;
	}
	#gNBTopoConfig .circleTip {
		width: 6px;
		height: 6px;
		background: #333333;
		border-radius: 50%;
		margin: 8px 6px 0 0;
	}

	#gNBTopoConfig .cuWarp .el-input__suffix{
		height: 26px;
		display: flex;
		align-items: center;
	}
	#gNBTopoConfig .errorBoxCls {color: #FA5555; font-size: 12px; margin-top: -6px;} 
	#gNBTopoConfig .itemListBoxCls{
		margin-left: 188px;
	}
	#gNBTopoConfig .itemCls{
		padding: 0px 5px;
		height: 24px;
		display: inline-block;
		line-height: 24px;
		border: 1px solid #a0c4f9;
        color:var(--main-color);
		background: rgba(var(--main-color-rgba1),0.1);
		border-color:var(--main-color) !important;
		margin-top:4px;
		border-radius: 2px;
		margin-right: 10px;
		white-space: nowrap;
	}
	#gNBTopoConfig .itemCls .ipText{
		font-size: 12px;
		margin-left: 10px;
		display: inline-block;
	}
	#gNBTopoConfig .itemCls .ipDelete{ margin-top:6px;}
	#gNBTopoConfig .itemCls .el-icon-close:before{
		content:"\e778";
		
	}
	#gNBTopoConfig .itemListBoxCls .el-icon-close{
		font-size: unset;
		position: unset;
		top: unset;
		right: unset;
	}
	#gNBTopoConfig .el-form-item__error{padding-top: 0px;}
	/* #gNBTopoConfig .disabledIconBox .el-icon::before{
		color: #e9e9e9;
	} */
	#gNBTopoConfig .el-form-item__label{ line-height: 26px;}
	#gNBTopoConfig .el-tab-pane{overflow: auto; border: 0 !important;}
	#gNBTopoConfig .el-collapse{border: none;}
	#gNBTopoConfig .nguIpWarp .el-input__suffix{
		top: 6px;		
	}
	#gNBTopoConfig .el-collapse-item__content{ padding-bottom: 0px;}
	#gNBTopoConfig .el-collapse-item__wrap { border-bottom: none;}
	#gNBTopoConfig .btn-next .el-icon-arrow-right{ font-size:12px !important;}
	#gNBTopoConfig .btn-next .el-icon-arrow-right:before{ content:"\e794" !important;color:#c0c4cc !important;}

	
.defaultTabs .el-tabs__header {
	width: 100% !important;
	border-bottom: 1px solid #D5DCEC !important;
}
.defaultTabs .el-tabs__content {
	width: 100% !important;
	background: #FFFFFF !important;
}
.defaultTabs .el-tabs__item {
	border: 0 !important;
}
.defaultTabs .el-tabs__active-bar {
	display: unset;
}
.defaultTabs .el-tabs__item.is-active {
	background-color: unset;
}
/*
.defaultTabs .el-tabs__item {
	height: 40px; 
	line-height: 40px;
	border-bottom: 1px solid #D5DCEC;
}
.defaultTabs .el-tabs__item.is-active {
	background-color: rgba(var(--main-color-rgba1),0.08);
}


*/
</style>

<div class="commonWarp" id="gNBTopoConfig" style='width: 100%; height: 100%; background: #FFFFFF; position: relative;'>	
	<el-tabs v-model='activeName' class="h100 defaultTabs" @tab-click='tabClick'>		
		<!--TOPO  -->
		<el-tab-pane name='topo' label="TOPO">
			<div id="myDiagramDiv" class="topoContent"></div>
		</el-tab-pane>
		
		<!-- Cell Management-->
		<el-tab-pane name='cell' label="Cell Management" v-if="optBtnShow">
			<!-- BBU: CU、DU	 -->
			<el-form ref="settingForm" :model="settingForm" :rules="formRules" label-position="left" label-width="160px">
				<el-collapse v-model="activeNames">
					<!-- Basic -->
					<el-collapse-item name="basic">
						<template slot='title'>
							<div style="display:inline-block;margin-left:70px;">
								<span class="title-icon" style="vertical-align:sub"></span>
								<span style="font-size:14px;font-weight:bold">BBU</span>
							</div>							
						</template>
						<div style="padding-left: 88px;" class="cuWarp"> 
							<div class="titleWarp" style="margin-top: 10px;">
		 						<div class="circleTip"></div>
		 						<span>CU</span>
		 					</div>
		 					<div style="display: flex;">
		 						<el-form-item prop="f1apLocalIp" label="F1AP Local IP" style="margin:0px 0px 30px 26px; width: 45%" >
									<el-input style='width:150px;' v-model="settingForm.f1apLocalIp" :disabled="isabled"></el-input>
								</el-form-item>
			 					<el-form-item prop="f1uIp" label="F1U IP" style="margin:0px 0px 30px 0;">
									<el-input style='width:150px;' v-model="settingForm.f1uIp" :disabled="isabled"></el-input>
								</el-form-item>
		 					</div>
		 					
		 					<div style="display: flex;">
								<div style="width: 45%;">
									<el-form-item prop="amfIp" label="AMF IP" style="margin:0px 0px 0 26px;">
										<el-input style='width:150px;' v-model="settingForm.amfIp" :disabled="isabled">
											<i slot="suffix" class="el-icon el-icon-plus" @click="addAmfIp" v-show="settingForm.amfIp"></i>
											<div slot="suffix" class="disabledIconBox">
												<i  class="el-icon el-icon-plus" v-show="!settingForm.amfIp"></i>
											</div>
										</el-input>
										<p class="errorBoxCls">{{amfIpErrorMessage}}</p>
									</el-form-item>
									<div class="itemListBoxCls">
										<div v-for="item in amfIpList" class="itemCls">
											<span class="ipText">{{item.ip}}</span>
											<span class="el-icon el-icon-close ipDelete" @click="amfIpListDel(item)"></span>
										</div>
									</div>
								</div>
								
								<div style="width: 50%;">
									<el-form-item prop="ngapLocalIp" label="NGAP Local IP" style="margin:0px 0px 0 26px">
										<el-input style='width:150px;' v-model="settingForm.ngapLocalIp" :disabled="isabled">
											<i slot="suffix" class="el-icon el-icon-plus" @click="addNgapLocalIp" v-show="settingForm.ngapLocalIp"></i>
											<div slot="suffix" class="disabledIconBox">
												<i  class="el-icon el-icon-plus" v-show="!settingForm.ngapLocalIp"></i>
											</div>
										</el-input>
										<p class="errorBoxCls">{{ngapLocalIpErrorMessage}}</p>
									</el-form-item>
									<div class="itemListBoxCls">
										<div v-for="item in ngapLocalIpList" class="itemCls">
											<span class="ipText">{{item}}</span>
											<span class="el-icon el-icon-close ipDelete" @click="ngapLocalIpListDel(item)"></span>
										</div>
									</div>
								</div>
							</div>
		 					
		 					<div style="display: flex;">
		 						<div style="width: 45%; margin-top: 30px;" class="nguIpWarp">
									<el-form-item prop="nguIp" label="NGU IP" style="margin-left: 26px; margin-bottom: 0px; ">
										<el-input style='width:150px;' v-model="settingForm.nguIp"  placeholder="" :disabled="isabled">
											<i slot="suffix" class="el-icon el-icon-plus" @click="addNguIp" v-show="settingForm.nguIp"></i>
											<div slot="suffix" class="disabledIconBox" style="margin-bottom: 14px">
												<i  class="el-icon el-icon-plus" v-show="!settingForm.nguIp"></i>
											</div>
										</el-input>
										<p class="errorBoxCls">{{nguIpErrorMessage}}</p>
									</el-form-item>
									<div class="itemListBoxCls" >
										<div v-for="item in nguIpList" class="itemCls">
											<span class="ipText">{{item}}</span>
											<span class="el-icon el-icon-close ipDelete" @click="nguIpListDel(item)"></span>
										</div>
									</div>
								</div>
							</div>
							
							
		 					<!-- DU -->
		 					<div class="titleWarp" style="margin-top: 24px;">
		 						<div class="circleTip"></div>
		 						<span>DU</span>
		 					</div>
		 					<div style="display: flex;">
		 						<el-form-item prop="f1apLocalIp2" label="F1AP Local IP" style="margin:0px 0px 0px 26px; width: 45%" >
									<el-input style='width:150px;' v-model="settingForm.f1apLocalIp2" :disabled="isabled"></el-input>
								</el-form-item>
			 					<el-form-item prop="f1uIp2" label="F1U IP">
									<el-input style='width:150px;' v-model="settingForm.f1uIp2" :disabled="isabled"></el-input>
								</el-form-item>
		 					</div>
						</div>
					</el-collapse-item>
				</el-collapse>
			</el-form>
			<div style="border-top: 1px solid #e9e9e9; margin-top: 1px;">
				<el-collapse v-model="activeNameHub">
					<!-- HUB -->
					<el-collapse-item name="hub">
						<template slot='title'>
							<div style="display:inline-block;margin-left:70px;">
								<span class="title-icon" style="vertical-align:sub"></span>
								<span style="font-size:14px;font-weight:bold">HUB</span>
							</div>							
						</template>					
						<div style="height: 200px; background: #eee;margin-left: 70px;margin-top: 10px; margin-right: 60px;">
							<el-ctable id="euTable" ref="euTable" :url="euUrl" :query-params="queryParams" :height="height" pagination="true" rownumber="true">
								<el-table-column label="" width="30" class-name="no-text-tips" v-if="!isabled">
									<template slot-scope="scope">
										<div class="el-icon el-icon-operation-more operationTip" @click="optClick(scope.row,event)" v-clickoutside="handerClose"></div>
									</template>
								</el-table-column>
								<el-table-column label="<%=rb.getString("LuYouSuoYin") %>" prop="RouteIndex" sortable="true"></el-table-column>
								<el-table-column label="<%=rb.getString("ZhuangTai") %>" prop="Status">
									<template slot-scope="scope">
										<div v-if="scope.row.Status == '1'">
											<span class='el-icon el-icon-status-conn-on euStatus'></span><span class="statusTip"><%=rb.getString("ZhengChang") %></span>
										</div>
										<div v-if="scope.row.Status == '2'">
											<span class='el-icon el-icon-status-conn-off euStatus'></span><span class="statusTip"><%=rb.getString("LiXian") %></span>
										</div>
										<div v-if="scope.row.Status == '3'">
											<span class='el-icon el-icon-status-alarm euStatus'></span><span class="statusTip"><%=rb.getString("GaoJingGuanLi") %></span>
										</div>							
									</template>
								</el-table-column>
								<el-table-column label="<%=rb.getString("XiaoZhanBianMa") %>" prop="SerialNumber"></el-table-column>
								<el-table-column label="<%=rb.getString("MoKuaiXingHao") %>" prop="ModelName"></el-table-column>
								<el-table-column label="<%=rb.getString("RuanJianBanBen") %>" prop="SoftwareVersion"></el-table-column>
							</el-ctable>
							<el-cmenu ref="menu_eu" :data="menus_eu" @click="clickMenuEu"></el-cmenu>
						</div>
					</el-collapse-item>	
				</el-collapse>
			</div>
			
			<!-- RRU -->
			<div style="border-top: 1px solid #e9e9e9; margin-top: 1px;">
				<el-collapse v-model="activeNameRru">
				
					<el-collapse-item name="rru">
						<template slot='title'>
							<div style="display:inline-block;margin-left:70px;">
								<span class="title-icon" style="vertical-align:sub"></span>
								<span style="font-size:14px;font-weight:bold">RRU</span>
							</div>							
						</template>
						
						<div style="height: 200px; background: #eee;margin-left: 70px;margin-top: 10px; margin-right: 60px;">
							<el-ctable id="ruTable" ref="ruTable" :url="ruUrl" :query-params="queryParams"
								 :height="height" pagination="true" rownumber="true">
								<el-table-column label="" width="30" class-name="no-text-tips" v-if="!isabled">
									<template slot-scope="scope">
										<div class="el-icon el-icon-operation-more operationTip" @click="optClickRu(scope.row,event)" v-clickoutside="handerClose"></div>
									</template>
								</el-table-column>
								<el-table-column label="<%=rb.getString("LuYouSuoYin") %>" prop="RouteIndex" sortable="true"></el-table-column>
								<el-table-column v-if="false" label="<%=rb.getString("XuLieHao") %>" prop="ruIndex"></el-table-column>
								<el-table-column label="<%=rb.getString("ZhuangTai") %>" prop="Status">
									<template slot-scope="scope">
										<div v-if="scope.row.Status == '1'">
											<span class='el-icon el-icon-status-conn-on euStatus'></span><span class="statusTip"><%=rb.getString("ZhengChang") %></span>
										</div>
										<div v-if="scope.row.Status == '2'">
											<span class='el-icon el-icon-status-conn-off euStatus'></span><span class="statusTip"><%=rb.getString("LiXian") %></span>
										</div>
										<div v-if="scope.row.Status == '3'">
											<span class='el-icon el-icon-status-alarm euStatus'></span><span class="statusTip"><%=rb.getString("GaoJingGuanLi") %></span>
										</div>
									</template>
								</el-table-column>
								<el-table-column label="<%=rb.getString("XiaoZhanBianMa") %>" prop="SerialNumber"></el-table-column>
								<el-table-column label="<%=rb.getString("XiaXingFaSheGongLvZengYi") %>" prop="TxGain"></el-table-column>
								<el-table-column label="<%=rb.getString("FangSheZhuangTai") %>" prop="RFTxStatus">
									<template slot-scope="scope">
										<div v-if="scope.row.RFTxStatus == '0'">
											<span class='el-icon el-icon-status-disable euStatus'></span><span class="statusTip"><%=rb.getString("GuanBi") %></span>
										</div>
										<div v-if="scope.row.RFTxStatus == '1'">
											<span class='el-icon el-icon-status-enable euStatus'></span><span class="statusTip"s><%=rb.getString("KaiQi") %></span>
										</div>
									</template>
								</el-table-column>
								<el-table-column label="<%=rb.getString("MoKuaiXingHao") %>" prop="ModelName"></el-table-column>
								<el-table-column label="<%=rb.getString("RuanJianBanBen") %>" prop="SoftwareVersion"></el-table-column>
							</el-ctable>
							<el-cmenu ref="menu_ru" :data="menus_ru" @click="clickMenuRu"></el-cmenu>
						</div>
					</el-collapse-item>
				</el-collapse>
			</div>
			
		</el-tab-pane>
		
		<div class='footer' style="border-top: 1px solid #D5DCEC; padding: 12px 0;" v-if="activeName == 'cell'">	
			<div style="margin-left: 30px;">
				<el-button type="primary" @click="formSubmit" :disabled="isabled"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeTopoPage"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</div>
	</el-tabs>	
</div>

<script type="text/javascript">
	var commonSn = '', regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
	//当前节点的数据
	var curNodeData = {};
	var gNBTopoConfig = new Vue({
		el:'#gNBTopoConfig',
		data(){	
			var vm = this;
			var validateIPv4AndIPv6 = (rule,value,callback)=>{
				if(value == '' || value == undefined || value == null){
					callback(new Error('<%=rb.getString("IPShuRuTiShi")%>'))
				}else{
					if(vm.isValidIP(value) || vm.isIPv6(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("IPShuRuTiShi")%>'))
					}
				}
			};
			return {
				activeName:'topo',
				height: '100%',
				euUrl: '',
				ruUrl: '',
				queryParams: {
					smallCellCode : ""
				},
				menus_eu: [],
				menus_ru: [],				
				rowDataEu: [],
				rowDataRu: [],
				operType: '',
				
				activeNames: ['basic'],
				activeNameHub: 'hub',
				activeNameRru: 'rru',
				settingForm: {
					f1apLocalIp: '',
					f1uIp: '',
					amfIp: '',
					ngapLocalIp:'',
					nguIp: '',
					
					f1apLocalIp2: '',
					f1uIp2: '',
				},
				oldSettingForm: {					
					f1apLocalIp: '',
					f1uIp: '',					
					amfIp: '',
					ngapLocalIp:'',
					nguIp: '',
					
					f1apLocalIp2: '',
					f1uIp2: '',
				},
				
				amfIpList: [],
				ngapLocalIpList: [],
				oldNgapLocalIpList: [],
				nguIpList: [],
				oldNguIpList: [],
				delAMFIpList: [],
				amfIpErrorMessage: '',
				ngapLocalIpErrorMessage: '',
				nguIpErrorMessage: '',
				formRules:{
					f1apLocalIp:[
						{validator: validateIPv4AndIPv6}
					],
					f1uIp:[
						{validator: validateIPv4AndIPv6}
					],
					f1apLocalIp2:[
						{validator: validateIPv4AndIPv6}
					],
					f1uIp2:[
						{validator: validateIPv4AndIPv6}
					]					
				},
				isOnline: false
			}
		},
		computed:{
			isabled() {
				return writableMap['CODE_GNB'] == false || this.isOnline == false;
			},
			optBtnShow() {
				return writableMap['CODE_GNB'] == true;
			},
		},
		methods:{
			initTask(sn,status){
				var vm = this;
				//commonSn = sn;
				commonSn = gnbTabSettingVue.rowData.small_cell_code;
				if(gnbTabSettingVue.rowData.connection_status == 'Off'){
					vm.isOnline = false;
				}else{
					vm.isOnline = true;
				}
				
				initTopo();
			},
			tabClick(tab){
				var vm = this;	
				if(tab.name == 'cell'){
					vm.queryParams.smallCellCode = commonSn;
					vm.euUrl = '${ctx}/gnb/gnbMonitor/getEUInfos.action';
					vm.ruUrl = '${ctx}/gnb/gnbMonitor/getRUInfos.action';
					//获取BBU:ＣＵ、DU 数据
					var params={
							smallCellCode: commonSn
						};
					axios.post('${ctx}/gnb/gnbMonitor/getCellManageInfos.action',stringify(params)).then(function(response){
						var data = response.data;
						if(data){
							var cuInfo = data.cuInfo, duInfo = data.duInfo;
							
							vm.settingForm.f1apLocalIp = data.cuInfo.f1apLocalIp;
							vm.settingForm.f1uIp = data.cuInfo.f1uIp;
							
							vm.settingForm.f1apLocalIp2 = data.duInfo.f1apLocalIp;
							vm.settingForm.f1uIp2 = data.duInfo.f1uIp;

							if(cuInfo.amfIpList){
								vm.amfIpList = cuInfo.amfIpList;
							}
							if(cuInfo.ngapLocalIpList){
								vm.ngapLocalIpList = cuInfo.ngapLocalIpList.split(',');
								vm.oldNgapLocalIpList = cuInfo.ngapLocalIpList.split(',');
							}
							if(cuInfo.nguIpList){
								vm.nguIpList = cuInfo.nguIpList.split(',');
								vm.oldNguIpList = cuInfo.nguIpList.split(',');
							} 
							//对比参数是否发生变化
							vm.oldSettingForm.f1apLocalIp = data.cuInfo.f1apLocalIp;
							vm.oldSettingForm.f1uIp = data.cuInfo.f1uIp;
							
							vm.oldSettingForm.f1apLocalIp2 = data.duInfo.f1apLocalIp;
							vm.oldSettingForm.f1uIp2 = data.duInfo.f1uIp;							
						}
					}).catch(function(error){})
				}
			},
			addAmfIp(){
				var vm = this, val = vm.settingForm.amfIp,
				params={
					ip: vm.settingForm.amfIp
				};
				if(val){
					if(vm.isValidIP(val) || vm.isIPv6(val)) {
						var result = vm.amfIpList.some(item=>item.ip == val);
						if(result){
							vm.amfIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							vm.amfIpList.push(params);
							vm.settingForm.amfIp = '';
							vm.amfIpErrorMessage = '';
						}
					}else {
						vm.amfIpErrorMessage = '<%=rb.getString("IPGeShiBuDui")%>';
					}
				}
			},
			amfIpListDel(row){
				var vm = this;
				if(row.configIndex){
					vm.delAMFIpList.push(row.configIndex);
				}
				vm.amfIpList = vm.amfIpList.filter((items)=>{
					return items.ip != row.ip
				})
			},
			addNgapLocalIp(){
				var vm = this, val = vm.settingForm.ngapLocalIp;

				if(val){
					if(vm.isValidIP(val) || vm.isIPv6(val)) {
						var result = vm.ngapLocalIpList.some(item=>item == val);
						
						if(result){
							vm.ngapLocalIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							vm.ngapLocalIpList.push(val);
							vm.settingForm.ngapLocalIp = '';
							vm.ngapLocalIpErrorMessage = '';							
						}
					}else {
						vm.ngapLocalIpErrorMessage = '<%=rb.getString("IPGeShiBuDui")%>';
					}
				}
			},
			ngapLocalIpListDel(delIp){
				var vm = this;
				vm.ngapLocalIpList = vm.ngapLocalIpList.filter((items)=>{
					return items != delIp
				})
			},
			addNguIp(){
				var vm = this, val = vm.settingForm.nguIp;
				if(val){
					
					if(vm.isValidIP(val) || vm.isIPv6(val)) {
						var result = vm.nguIpList.some(item=>item == val);
						if(result){
							vm.nguIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							vm.nguIpList.push(val);
							vm.nguIpErrorMessage = '';
							vm.settingForm.nguIp = '';
						}
					}else {
						vm.nguIpErrorMessage = '<%=rb.getString("IPGeShiBuDui")%>';
					}
				}							
			},
			nguIpListDel(delIp){
				var vm = this;
				vm.nguIpList = vm.nguIpList.filter((items)=>{
					return items != delIp
				})
				
			},
			//Cell Management 提交
			formSubmit(){
				var vm = this, isChanged = false;
				if(vm.amfIpList.length == 0){
					vm.amfIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
					isChanged = true;
				}
				if(vm.ngapLocalIpList.length == 0){
					vm.ngapLocalIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
					isChanged = true;
				}
				if(vm.nguIpList.length == 0){
					vm.nguIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
					isChanged = true;
				}				
                
				var params = {}, addAmfIpArr = [], cuInfo = {}, duInfo = {};
				// 只提交发生变化的参数
				vm.amfIpList.map((item,index)=>{
					if(!item.configIndex){
						addAmfIpArr.push(item)
					}
				});

				//判断参数是否发生变化
				if(vm.settingForm.f1apLocalIp == vm.oldSettingForm.f1apLocalIp ){
					cuInfo.f1apLocalIp = '';
				}else{
					cuInfo.f1apLocalIp = vm.settingForm.f1apLocalIp;
				}
				if(vm.settingForm.f1uIp == vm.oldSettingForm.f1uIp ){
					cuInfo.f1uIp = '';
				}else{
					cuInfo.f1uIp = vm.settingForm.f1uIp;
				}
				
				if(vm.ngapLocalIpList.join(',') == vm.oldNgapLocalIpList.join(',') ){
					cuInfo.ngapLocalIpList = '';
				}else{
					cuInfo.ngapLocalIpList = vm.ngapLocalIpList.join(',');
				}
				if(vm.nguIpList.join(',') == vm.oldNguIpList.join(',') ){
					cuInfo.nguIpList = '';
				}else{
					cuInfo.nguIpList = vm.nguIpList.join(',');
				}
				
				if(vm.settingForm.f1apLocalIp2 == vm.oldSettingForm.f1apLocalIp2 ){
					duInfo.f1apLocalIp = '';
				}else{
					duInfo.f1apLocalIp = vm.settingForm.f1apLocalIp2;
				}
				if(vm.settingForm.f1uIp2 == vm.oldSettingForm.f1uIp2 ){
					duInfo.f1uIp = '';
				}else{
					duInfo.f1uIp = vm.settingForm.f1uIp2;
				}
				// 提交参数
				params.smallCellCode = commonSn;
				cuInfo.amfIpList = addAmfIpArr;
				params.cuInfo = JSON.stringify(cuInfo);	
				params.duInfo = JSON.stringify(duInfo);
				params.delAMFIpList = vm.delAMFIpList.join(',');
				
				vm.$refs.settingForm.validate((valid) => {
                    if (valid && isChanged == false) {
                    	axios.post("${ctx}/gnb/gnbMonitor/updateCellManageInfos.action",stringify(params)).then(function(response){
        					var data = response.data;
        					if(data["success"]){
        						vm.$message({
        							message:'<%=rb.getString("ChengGong")%>',
        							type:'success',
        						});       						
        						// 设备同步
        						var params = {smallCellCode: commonSn};
        			            
        			            $.post( "${ctx}/cell/param/refreshCellInfo.action", params, function(data){
        			                if (data["success"]) {
        			                } else {
        			                	vm.$message.error(data["message"])
        			                }
        			            }, "json");
        					}else{
        						vm.$message.error(data["message"])
        					}
        					//成功之后关掉这个页面
        					vm.closeTopoPage();
        				})                  
                    }
                })								
			},
					
			optClick(row,ev){
				var vm = this, euRebootFlag = false;
				vm.rowDataEu = row;
				vm.operType = 'eu';
				//Status: 1-在线，2离线
				if(row.Status == '2') {
					euRebootFlag = false;
				}else{
					euRebootFlag = true;
				}
				vm.menus_eu = [
					{label:'<%=rb.getString("ChongQi")%>',cls:'el-icon el-icon-operation-reboot',code:'reboot',disable:!euRebootFlag}
				]
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu_eu.show(ev);
				})
			},
			//EU
			clickMenuEu(ev){
				var codes = {
					reboot:this.reboot
				}
				if(codes[ev.code]){
					codes[ev.code]();
				}
			},
			//RU
			optClickRu(row,ev){
				var vm = this, ruRebootFlag = false, rftxFlag = false;
				vm.rowDataRu = row;
				vm.operType = 'ru';
				var rfText = '';
				var rfIcon = '';
				if(row.Status == '2') {
					ruRebootFlag = false;
					rftxFlag = false;
				}else{
					ruRebootFlag = true;
					rftxFlag = true;
				}
				if(row.RFTxStatus == '0'){
					rfText = '<%=rb.getString("ShePinCaoZuo")%> <%=rb.getString("Kai")%>';
					rfIcon = 'el-icon el-icon-operation-enable1';
				}else{
					rfText = '<%=rb.getString("ShePinCaoZuo")%> <%=rb.getString("Guan")%>';
					rfIcon = 'el-icon el-icon-operation-disable1';
				}
				vm.menus_ru = [
					{label:'<%=rb.getString("ChongQi")%>',cls:'el-icon el-icon-operation-reboot',code:'reboot',disable:!ruRebootFlag},
					{label:rfText,cls:rfIcon,code:'rf',disable:!rftxFlag}
				]
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu_ru.show(ev);
				})
			},
			//点击表格菜单
			clickMenuRu(ev){
				var codes = {
					reboot:this.reboot,
					rf:this.changeRfRu
				}
				if(codes[ev.code]){
					codes[ev.code]();
				}
			},
			//隐藏表格-操作菜单
			handerClose(){
				this.$refs.menu_eu.hide();
				this.$refs.menu_ru.hide();
			},
			//关闭当前Slide
			closeTopoPage(){
				//gnbMonitor.$refs.topoCellManage.hide();

				eventBus.$emit('close-gnb-settingPage');
			},			
			//重启
			reboot(){
				var vm = this, table, params = {};
				
				if(vm.operType == 'eu'){
					params.smallCellCode = vm.rowDataEu.smallCellCode;
					params.rebootStatus = '1';
					params.euIndex = vm.rowDataEu.euIndex;
					params.ruIndex = '';
					params.type = 'eu';
					params.serialNumber = vm.rowDataEu.SerialNumber;
					table = vm.$refs.euTable;
				}else{
					params.smallCellCode = vm.rowDataRu.smallCellCode;
					params.rebootStatus = '1';//重启开关： 0-不重启，1-重启
					params.euIndex = vm.rowDataRu.euIndex;
					params.ruIndex = vm.rowDataRu.ruIndex;
					params.type = 'ru';
					params.serialNumber = vm.rowDataRu.SerialNumber;					
					table = vm.$refs.ruTable;
				}
				var confirmStr = '<%=rb.getString("QueDingChongQiSheBei")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:"warningConfirm",
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post('${ctx}/gnb/gnbMonitor/updateEUOrRUReboot.action',stringify(params)).then(function(response){
	    				var data = response.data;
	    				if(data["success"]){
							vm.$message({
								type:'success',
								message:'<%=rb.getString("ChengGong")%>'
							})
							table.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					}).catch(function(error){}) 
				})
			},
			//射频操作
			changeRfRu(){
				var vm = this, params = {};
					params.smallCellCode = vm.rowDataRu.smallCellCode;
					params.euIndex = vm.rowDataRu.euIndex;
					params.ruIndex = vm.rowDataRu.ruIndex;
					params.serialNumber = vm.rowDataRu.SerialNumber;
				
				if(vm.rowDataRu.RFTxStatus == '0'){
					params.rfTxStatus = '1'
				}else{
					params.rfTxStatus = '0'
				}
				axios.post('${ctx}/gnb/gnbMonitor/updateRFTxStatus.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$message({
							type:'success',
							message:'<%=rb.getString("ChengGong")%>'
						})
						vm.$refs.ruTable.refresh();
					}else{
						vm.$message.error(data["message"])
					}
				}).catch(function(error){}) 
			},
			delNodeClick(){
				var vm = this;
				$.messager.confirm('<%=rb.getString("QueRen")%>', '<%=rb.getString("QueDingShanChuSheBei")%>', function (r) {
					if (r) {
						var param = {
							smallCellCode: commonSn,
							serialNumber: curNodeData.serialnumber,
							text: curNodeData.text
						};
						$.post("${ctx}/gnb/gnbMonitor/delNodeForTOPOInfo.action", param, function (data) {
							if (data["success"]) {
								vm.$message({
		    						message: '<%=rb.getString("ChengGong")%>',
		    						type: 'success'
		    					});
								//更新TOPO
								myDiagram.div = null;
								initTopo();								
							}else{
								vm.$message.error(data["message"]);
							}
						}, "json");
					}
				}).addClass("seriousConfirm");
			},
			//校验IP
			isValidIP(ip){
				var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
				return reg.test(ip);     
			},
			//Ipv6校验 
			isIPv6(str){ 
				var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
				return reg.test(str);
			},
		},		
		mounted(){
			this.initTask();
			//eventBus.$off("show-topo").$on("show-topo",this.initTask);
		}	
	});
	
	
	//画图： 图表由 节点、文字、线组成
	//1、stroke: 边框颜色；  2、 margin: 边框间距,   margin: new go.Margin(10,20,30,40) 外边距;   3、fill: 背景颜色；
	//1、TextBlock: 创建文本；   2、Shape: 创建图形；  3、 Node:节点（结合文本与图形）；  4、Links 连线
	function initTopo(){
		// 创建图表
		var $ = go.GraphObject.make;
		//绑定DOM元素
		myDiagram = 
			$(go.Diagram,'myDiagramDiv',
				{
					isReadOnly: true,					
					// 画布的位置设置：居中显示内容, 不可拖动画布
	            	contentAlignment: go.Spot.Center,
	            	// 启用Ctrl-Z和Ctrl-Y撤销重做功能
					'undoManager.isEnabled': false,
					//去掉节点 点击时的边框颜色
					nodeSelectionAdornmentTemplate:
						$(go.Adornment,'Aiuto',
							$(go.Shape, 'Rectangle',{fill:'white',stroke: null})		
						),
					//树形布局排列方式，从上到下（0，90，180，270） 、每层间距,
					layout: $(go.TreeLayout,
						{angle: 90, layerSpacing: 66}) 					
				}
			);
		
		//新建节点
		myDiagram.nodeTemplate = 
			$(go.Node, 'Auto',
				//突出显示 鼠标滑过、离开
				{
					selectionAdorned: false,
					selectionChanged: allChanged,
					//鼠标滑过显示 设备名称
					mouseEnter: mouseEnter,
					mouseLeave: mouseLeave
				},
				//节点禁止拖动
				{movable: false},
				
				//设置节点形状：圆角矩形
				$(go.Shape, 'RoundedRectangle',
					//设置大小、边框大小、颜色、背景色、鼠标手势
					{width: 120, height: 60, strokeWidth: 2, margin: new go.Margin(0,0,0,20), cursor: 'grab', name: 'SHAPE'},
					//将节点数据nodeDataArray   .color与节点背景色建立联系
					//绑定背景色
					new go.Binding('fill', 'bgColor'),
					//绑定边框色
					new go.Binding('stroke', 'borderColor'),
				),
				//设置文本节点
				$(go.TextBlock, textStyle(),
					//设置文本样式：大小，是否换行，margin
					{ wrap: go.TextBlock.WrapFit, name: 'TEXT'},
					//将TextBlock.text 绑定到 Node.data.text
					new go.Binding('text', 'text'),
					new go.Binding('cursor', 'cursor')),
				//添加 tooltip，显示节点对应的基站编码
				{toolTip:
					$("ToolTip",
						$(go.TextBlock,{margin: 6},
							new go.Binding('text','serialnumber', 
								function(sn){ 
									return '<%=rb.getString("KPISheBei")%>: ' + sn;
								}							
							))		
					)					
				} 
			);
		
		function allChanged(node){
			if(node.isSelected){
				if(node.part.data.outOfContact == 'true'){
				    selectionAdornment.adornedObject = node;
				    node.addAdornment('Radial',selectionAdornment);
			   }else{
				   var oldnode = selectionAdornment.adornedPart;
				   if(oldnode) oldnode.removeAdornment('Radial');
				   selectionAdornment.adornedObject = null;
			   } 
			}else{
				 var oldnode = selectionAdornment.adornedPart;
				 if(oldnode) oldnode.removeAdornment('Radial');
				 selectionAdornment.adornedObject = null;
			}
		};
		
		var selectionAdornment = 
			$(go.Adornment, 'Spot',
				$(go.Panel, 'Auto',
					$(go.Shape, {fill: null, stroke:'#DCDFE6', strokeWidth: 0}),
					$(go.Placeholder)
				), 
				$('Button',
					{ alignment: go.Spot.Top, alignmentFocus: go.Spot.Left, 
						width: 140, cursor: 'pointer', padding: go.Margin.parse('0 0 20 10'), 
						'ButtonBorder.fill': '#FFFFFF',
						'ButtonBorder.stroke': '#E9E9E9',
						'_buttonFillOver': '#EDF6FF',
						'_buttonStrokeOver': '#E9E9E9',
						'_buttonFillFocus': '#EDF6FF',
						'_buttonStrokeFocus': '#E9E9E9',
						click: function(e, obj){
							if(obj.part.data.outOfContact == 'true'){
							    curNodeData = obj.part.data;
							    gNBTopoConfig.delNodeClick();
						    }else{
							   return;
						    }
						}
					},
					
					$(go.TextBlock, '<%=rb.getString("ShanChu") %>',
							{margin: 6,stroke: '#333333'},		
					)							
				)	
			)
				
		//虚线 实线	 连接	
		var templmap = new go.Map(), color  = '#FA5555';
		var defaultTemplate = 
			$(go.Link,
				$(go.Shape, { stroke: color, strokeWidth: 2})
			)
		var dashedTemplate = 
			$(go.Link,
				$(go.Shape, { stroke: color, strokeWidth: 2, strokeDashArray: [6, 3]})			
			)
		
		templmap.add('',defaultTemplate);
		templmap.add('dashed',dashedTemplate);
		myDiagram.linkTemplateMap = templmap; 
		
		//设置线条，暂无箭头
		myDiagram.linkTemplate = 
			$(go.Link,
				//线条连接样式：直线
				{curve: go.Link.Bezier},
				//线的连接形状
				$(go.Shape,
					{strokeWidth: 2, stroke: "#707070"}	
				)
			);
		
		 // 定义图形上的文字风格		 
	    function textStyle() {
	        return {         
	        	//文本颜色
	            stroke: "#333333",
	            font: "bold 12px normal"
	        }
	    };
	    //鼠标滑过修改背景颜色
		function mouseEnter(e,obj){			
			var shape = obj.findObject('SHAPE');
				if(obj.data.hoverColor == ''){
					shape.fill = '#F3F3F3';
				}else{
					shape.fill = obj.data.hoverColor;
				}
		};
		//鼠标离开还原色值
		function mouseLeave(e,obj){
			var shape = obj.findObject('SHAPE');
			shape.fill = obj.data.bgColor;
			
		};
	    
	   //获取TOPO 图数据
	   setTimeout(function(){
		  var params = { smallCellCode: commonSn};
		  axios.post('${ctx}/gnb/gnbMonitor/getTOPOInfo.action',stringify(params)).then(function(response){
		   		var data = response.data;
				if(data){
					//添加节点数据
					var nodeDataArray = data.nodeDataArray;
					
					//添加连接线数据
					var linkDataArray = data.linkDataArray;
	
					//新建关系图: 通过节点数据和关系数组完成关系图
					myDiagram.model = new go.GraphLinksModel(nodeDataArray,linkDataArray);	
				}
			}).catch(function(error){}) 

			/* var linkDataArray = [
				{from: 0, to: -1},
				{from: -1, to: 11, category:'dashed'},
				{from: -1, to: 13, category:'dashed'},
				{from: -1, to: 14, category:'dashed'}
			]; 
			var nodeDataArray = [		    
				{key: 0, text: 'BBU', bgColor: '#F2F6FF', borderColor:'#4D84FF', hoverColor: '#DCEBFE',},			
				{key: -1, text: 'HUB1', bgColor: '#F5FFF9', borderColor:'#67D972', hoverColor: '#D8F3E5', serialnumber: '1202000024019APP0061'},
				{key: 11, text: 'RRU1', bgColor: '#F3F3F3', borderColor:'#DCDFE6',  hoverColor: '#F3F3F3',serialnumber: '1202000024019APP0062', outOfContact: 'true'},
				{key: 13, text: 'HUB1-1', bgColor: '#F3F3F3', borderColor:'#DCDFE6', hoverColor: '#F3F3F3', serialnumber: '1202000024019APP0063', outOfContact: 'true'},
				{key: 14, text: 'HUB1-2', bgColor: '#F3F3F3', borderColor:'#DCDFE6',  hoverColor: '', serialnumber: '1202000024019APP0064', outOfContact: 'true'}
			];

			myDiagram.model = new go.GraphLinksModel(nodeDataArray,linkDataArray);*/
	   },500);	   
	   
	}
</script>

