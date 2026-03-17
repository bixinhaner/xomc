<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#deviceRecycleBinPage{
		background-color: #FFFFFF;
		height: 100%;
		width: 100%;
		overflow: auto;
		position: relative;
	}
	#deviceRecycleBinPage .loading::before{
		background-color: rgba(255,255,255,1);
	}
	#deviceRecycleBinPage .deviceRecycleBinHeaderCls{
		position:relative;
	}
	#deviceRecycleBinPage .el-ctable-toolbar{
		padding: 0px!important;
	}
	#deviceRecycleBinPage .deviceListTableCls{
		width: 100%;
		height: 100%;
		overflow: auto;
		position:relative;
	}
</style>
<div id='deviceRecycleBinPage'>
	<div class="container commonWarp" style="min-width: 900px;">
		<div class="deviceListTableCls">
			<el-ctable
				id="deviceTable"
				ref="ctableDevice"
				:url="deviceUrl"
				:height="height"
				:row-key="deviceTableRowKey"
				:query-params="params_device"
				pagination="true"
				:rownumber=true
				:limit="limitBatch"
				@selection-change='batchSelect'
				>
                <template slot="toolbar">
                    <div class="deviceRecycleBinHeaderCls">
                        <div class="toolbarHeadBtnBoxCls" style="height: 40px;">
                            <!-- 按钮   关闭 -->
                            <div class="newIconBoxCls-bt" style="position: absolute;right:20px" @click="closeDeviceRecycleBin" tip="<%=rb.getString("GuanBi")%>">		
                                <span class="el-icon-close el-icon"></span>
                            </div>
                            <div style="margin:0px 10px;font-size:14px;font-weight:bold;"><%=rb.getString("HuiShouZhan")%></div>
                            <div v-show="optDeviceShow" class="selectBlukBoxCls">
                                <div class="selectMain">
                                    <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                        <span class="el-icon-selected el-icon"></span>
                                        <span class="bulkSelectNumBoxCls">( {{deviceSelection.length}} )</span>
                                    </div>
                                    <div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectShow">
                                        <div class="selectBoxTitle">
                                            <span><%=rb.getString("YiXuan")%></span>
                                            <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
                                        </div>
                                        <div class="selectBoxMain">
                                            <div class="tableInfoCls">
                                                <div class="tableInfoHeader">
                                                    <div>{{selectedDialogTitle}}</div>
                                                    <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
                                                </div>
                                                <el-ctable
                                                    id="bulkSelectTable"
                                                    ref="bulkSelectTable"
                                                    :data="deviceSelection"
                                                    :showHeader="false"
                                                    :rownumber="false"
                                                    :front-pagination="true"
                                                    :row-key="deviceTableRowKey"
                                                    height="270px" pagination="true" >
                                                    <el-table-column prop="alarm_id" v-if="false"></el-table-column>
                                                    <el-table-column width="588">
                                                        <template slot-scope="scope" >
                                                            <div class="tableItemCls">
                                                                <span v-if="deviceType == 'eNB'">{{scope.row.serial_number}}</span>
                                                                <span v-if="deviceType == 'gNB'">{{scope.row.serial_number}}</span>
                                                                <span v-if="deviceType == 'CPE'">{{scope.row.macaddress}}({{scope.row.serial_number}})</span>
                                                                <span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
                                                            </div>
                                                        </template>
                                                    </el-table-column>
                                                </el-ctable>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <div v-show="optDeviceShow" :class="deviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="removeRecyleBin">
                                <span class="el-icon el-icon-operation-restore"></span>
                                <span><%=rb.getString("YiChuHuiShouZhan")%></span>
                            </div>
                            <div v-show="optDeviceShow" :class="deviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="deleteRecyleBinDevice">
                                <span class="el-icon el-icon-operation-delete"></span>
                                <span><%=rb.getString("PiLiangShanChu")%></span>
                            </div>
                        </div>
                        <div class="tableHeadBoxCls" style="position:relative;">
                            <div id="tableHeadQuery" class="tableHeadQueryBoxCls">
                                <div class="headQueryBox">
                                    <div class="queryGroup">
                                        <el-input v-model="searchText" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:260px;"></el-input>
                                        <i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
                                    </div>
                                </div>
                                <el-popfilter style="margin: 0 5px;"
                                    label='<%=rb.getString("SheBeiZu")%>'
                                    v-model="params_device.group_id"
                                    type="single"
                                    :list="deviceGroupOptions">
                                </el-popfilter>
                            </div>
                        </div>
                    </div>
                </template>
				<!--设备列表-->
				<el-table-column width="50" type="selection" prop="ck" v-if="optDeviceShow"></el-table-column>
				<el-table-column key="enbSn" v-if="deviceType == 'eNB'" label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="120" prop="serial_number"></el-table-column>
				<el-table-column key="enbCellName" v-if="deviceType == 'eNB'" label='<%=rb.getString("HostName")%>' min-width="120" prop="host_name"></el-table-column>
				<el-table-column key="enbMac" v-if="deviceType == 'eNB'" label='<%=rb.getString("MACDiZhi")%>' min-width="120" prop="mac_address"></el-table-column>
				<el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol !=='true'" label='<%=rb.getString("ChangShang")%>' min-width="120" prop="manufacturer"></el-table-column>
				<el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol !=='true'" label='<%=rb.getString("ChengShi")%>' min-width="100" prop="city"></el-table-column>
				<el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol !=='true'" label='<%=rb.getString("SuoShuZhiJu")%>' min-width="120" prop="sub_branches"></el-table-column>
				<el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol !=='true'" :label="siteNameLabel" min-width="100" prop="sub_station_name"></el-table-column>

				<el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" :label="siteNameLabel" min-width="140" prop="sub_station_name"></el-table-column>
				<el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" label='<%=rb.getString("AnZhuangXiangXiDiZhi")%>' min-width="200" prop="install_address"></el-table-column>
				<el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" label='<%=rb.getString("YeZhuLianXiFangShi")%>' min-width="100" prop="contact_number"></el-table-column>
				<el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" :label="siteIdLabel" min-width="100" prop="site_id"></el-table-column>
				<el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" label='Circuit Ref.' min-width="100" prop="circuit_ref"></el-table-column>
				<el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" label='Circuit J & O' min-width="100" prop="circuit_jo"></el-table-column>
				<el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" label='Status' min-width="100" prop="service_status"></el-table-column>
				<el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" label='Rom' min-width="100" prop="rom"></el-table-column>
				<el-table-column key="cpeSn" v-if="deviceType == 'CPE'" label='<%=rb.getString("CPEXuLieHao")%>' min-width="100" prop="serial_number"></el-table-column>
				<el-table-column key="cpeMac" v-if="deviceType == 'CPE'" label='<%=rb.getString("MACDiZhi")%>' min-width="100" prop="macaddress"></el-table-column>
				<el-table-column key="cpeImsi" v-if="deviceType == 'CPE'" label='IMSI' min-width="100" prop="imsi"></el-table-column>
				<el-table-column key="longitude" v-if="deviceType == 'eNB' || deviceType == 'CPE'" label='<%=rb.getString("JingDu")%>' min-width="100" prop="longitude"></el-table-column>
				<el-table-column key="latitude" v-if="deviceType == 'eNB' || deviceType == 'CPE'" label='<%=rb.getString("WeiDu")%>' min-width="100" prop="latitude"></el-table-column>
				<el-table-column key="height" v-if="deviceType == 'eNB' || deviceType == 'CPE'" label='<%=rb.getString("GaoDu")%>' min-width="100" prop="height"></el-table-column>
				<el-table-column key="cpeDistance" v-if="deviceType == 'CPE'" label='<%=rb.getString("JuLi")%>' min-width="100" prop="distance"></el-table-column>
				<el-table-column key="gnbSn" v-if="deviceType == 'gNB'" label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="100" prop="serial_number"></el-table-column>
				<el-table-column key="gnbMac" v-if="deviceType == 'gNB'" label='<%=rb.getString("MACDiZhi")%>' min-width="100" prop="mac_address"></el-table-column>
				<el-table-column key="offlineDays" prop="offlineDays" label='<%=rb.getString("LiXianTianShu")%>' min-width="120"></el-table-column>
				<el-table-column key="groupName" label='<%=rb.getString("SheBeiZuMingCheng")%>' min-width="100" prop="group_name"></el-table-column>
				<el-table-column key="moveType" label='<%=rb.getString("HuiShouFangShi")%>' min-width="100" prop="moveType">
					<template slot-scope="scope">
						<span v-if="scope.row.moveType === '1'"><%=rb.getString("CPEShouDong")%></span>
						<span v-if="scope.row.moveType === '0'"><%=rb.getString("CPEZiDong")%></span>
					</template>
				</el-table-column>
				<el-table-column key="moveTime" label='<%=rb.getString("HuiShouShiJian")%>' min-width="100" prop="moveTime"></el-table-column>
				<el-table-column key="move_author" label='<%=rb.getString("ZhangHu")%>' min-width="100" prop="move_author"></el-table-column>
			</el-ctable>
		</div>
	</div>
</div>
<script type="text/javascript">
var deviceRecycleBinVue = new Vue({
	el:'#deviceRecycleBinPage',
	data(){
		return {
            deviceType:'eNB',
			bulkSelectShow:false,
			searchText:'',
			placeholderText:'<%=rb.getString("QingShuRu")%>',
			deviceUrl:'',
			height:'100%',
			params_device:{
				searchText:'',
				group_id:'',
				timeZone:timeZone,
			},
			batchOperation:batchOperation,
			deviceSelection:[],
			isDeviceMoreParams:"",
			showOrHideCol : "",

			tabUrlList:{
				'eNB':'${ctx}/recycle/getRecycleDeviceListByPage.action?isGnb=0',
				'gNB':'${ctx}/recycle/getRecycleDeviceListByPage.action?isGnb=1',
				'CPE':'${ctx}/recycle/getCpeRecycleDeviceListByPage.action',
			},
			valCodes:{
				'eNB':'small_cell_code',
				'gNB':'small_cell_code',
				'CPE':'cpe_code',
			},
			deviceGroupOptions:[],
		}
	},
    computed:{
		limitBatch(){
			return this.batchOperation ? '' : 1;
		},
		selectedDialogTitle() {
			var vm = this,
				deviceType = vm.deviceType
				codes = {
					'eNB':'<%=rb.getString("Title_SheBeiBianMa")%>',
					'gNB':'<%=rb.getString("Title_SheBeiBianMa")%>',
					'CPE':'<%=rb.getString("MACDiZhi")%>+<%=rb.getString("CPEBianMa")%>',
				};

			return codes[deviceType];
		},
		deviceTableRowKey() {
			var vm = this,
				deviceType = vm.deviceType
				codes = {
					'eNB':'serial_number',
					'gNB':'serial_number',
					'CPE':'cpe_code',
				};

			return codes[deviceType];
		},
		optDeviceShow() {
			var vm = this,
				deviceType = vm.deviceType
				codes = {
					'eNB':'CODE_ENB_DEVICE_REGISTER',
					'gNB':'CODE_GNB_DEVICE_REGISTER',
					'CPE':'CODE_CPE_DEVICE',
				};

			return writableMap[codes[deviceType]] == true;
		},
		siteIdLabel(){
			return siteIdLabel
		},
		siteNameLabel(){
			return siteNameLabel
		},
	},
	methods:{
		// 初始化
		init(deviceType){
			var vm = this;
			vm.deviceType = deviceType;
			vm.isDeviceMoreParams = egwRegisterVue.isDeviceMoreParams;
			vm.showOrHideCol = egwRegisterVue.showOrHideCol;
			vm.deviceUrl = vm.tabUrlList[vm.deviceType];
			vm.queryDeviceGroupOption();
		},
		// 查询设备组
		queryDeviceGroupOption(){
			var vm = this;
			axios.post('${ctx}/system/deviceGroup/getSimpleDeviceGroupList.action',stringify({isAll:'0'})).then(function(response){
				let data = response.data
				if(data && data.length>0){
					var groupOptions = [{label:'All',value:''}];
					data.map((item,index)=>{
						var obj = {};
						obj.label = item.group_name;
						obj.value = item.id;
						groupOptions.push(obj);
					})
				}
				vm.deviceGroupOptions = groupOptions;
			}).catch(function(error){});
		},
		/**
		* 选择的批量数据
		* @param selection:传入批量数据对象
		*/
	    batchSelect(selection){
	    	var vm = this;
	    	vm.deviceSelection = selection;
	 	},
		// 打开已选弹窗
		openBulkSelectTable(){
			var vm = this;
			vm.bulkSelectShow = true
		},
		// 关闭已选弹窗
		closeBulkSelectTable(){
			var vm = this;
			vm.bulkSelectShow = false;
		},
		// 设备已选表格 清空事件
		clearBulkSelected(){
			var vm = this;

			vm.$refs.ctableDevice.clearSelection();
		},
		// 设备已选表格 单个删除事件
		delBulkSelected(rows){
			var vm = this,
				deviceType = vm.deviceType,
				codes = {
					'eNB':'small_cell_code',
					'gNB':'small_cell_code',
					'CPE':'cpe_code',
				},
				tabs = 'ctableDevice',
				rowKey = codes[deviceType];

			vm.deviceSelection = vm.deviceSelection.filter((items)=>{
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
        // 模糊搜索
		query(){
			var vm = this,
				deviceType = vm.deviceType,
				likeFields = {
					'eNB':'serial_number',
					'gNB':'serial_number',
					'CPE':'serial_number,macaddress',
				};
			vm.params_device.searchText = this.searchText;
			vm.params_device.like_fields = likeFields[deviceType];
		},
		// 搜索域聚焦事件
		queryInputFocus(){
			var vm = this,
				deviceType = vm.deviceType,
				codes={
					'eNB':'<%=rb.getString("Title_SheBeiBianMa")%>',
					'gNB':'<%=rb.getString("Title_SheBeiBianMa")%>',
					'CPE':'<%=rb.getString("CPEXuLieHao")%>/<%=rb.getString("MACDiZhi")%>',
				};

			vm.placeholderText = codes[deviceType];
		},
		// 搜索域失焦事件
		queryInputBlur(){
			var vm = this;
			vm.placeholderText = '<%=rb.getString("QingShuRu")%>';
		},
		// 移出设备回收站
		removeRecyleBin(){
			var vm = this,
	    		params = {},
				urlsCodes = {
					'eNB':'${ctx}/recycle/moveDeviceToSmallCellInfos.action',
					'gNB':'${ctx}/recycle/moveDeviceToSmallCellInfos.action?isGnb=1',
					'CPE':'${ctx}/recycle/moveDeviceToCpeInfos.action',
				},
				subCodes={
					'eNB':'smallCellCodeStr',
					'gNB':'smallCellCodeStr',
					'CPE':'cpeCodeStr',
				},
	    		urls = urlsCodes[vm.deviceType],
				idsList = [],
				deviceType = vm.deviceType;
			if(vm.deviceSelection.length <= 0)return
			vm.deviceSelection.map((item,index) => {
				if(deviceType == 'eNB' || deviceType == 'gNB'){
					idsList.push(item[vm.valCodes[deviceType]])
				}else{
					idsList.push(item[vm.valCodes[deviceType]]);
				}
			})
    	    params[subCodes[deviceType]] = idsList.join(',');
	    	vm.$confirm('<%=rb.getString("QueRenJiangSheBeiYiChuHuiShouZhan")%>','<%=rb.getString("QueRen")%>',{
				customClass:"warningConfirm",
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancalButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
			}).then(()=>{
				axios.post(urls,stringify(params)).then(function(response){
					let data = response.data;
					if ( data.success ){
						vm.$message({
					    	message: '<%=rb.getString("ChengGong")%>' ,
							type:'success',
						})
						vm.$refs.ctableDevice.refresh();
						vm.$refs.ctableDevice.clearSelection();
					}else {
						vm.$message.error(data.message)
					}
				}).catch(function(error){})
			}).catch(()=>{})
		},
		// 批量删除设备回收站设备
		deleteRecyleBinDevice(){
			var vm = this,
	    		params = {
					whereFrom:'recycle'
				},
				urlsCodes = {
					'eNB':'${ctx}/system/deviceGroup/delCellinfo.action',
					'gNB':'${ctx}/system/deviceGroup/delCellinfo.action?isGnb=1',
					'CPE':'${ctx}/cell/CPE/delCpeinfo.action',
				},
				subCodes={
					'eNB':'ids',
					'gNB':'ids',
					'CPE':'cpeCodes',
				},
	    		urls = urlsCodes[vm.deviceType],
				idsList = [],
				deviceType = vm.deviceType;
			if(vm.deviceSelection.length <= 0)return
			vm.deviceSelection.map((item,index) => {
				if(deviceType == 'eNB' || deviceType == 'gNB'){
					idsList.push(item[vm.valCodes[deviceType]] + '_' + item.product)
				}else{
					idsList.push(item[vm.valCodes[deviceType]]);
				}
			})
    	    params[subCodes[deviceType]] = idsList.join(',');

	    	vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
				customClass:"warningConfirm",
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancalButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning'
			}).then(()=>{
				axios.post(urls,stringify(params)).then(function(response){
					let data = response.data;
					if ( data.success ){
						vm.$message({
					    	message: '<%=rb.getString("ChengGong")%>' ,
							type:'success',
						})
						vm.$refs.ctableDevice.refresh();
						vm.$refs.ctableDevice.clearSelection();
					}else {
						vm.$message.error(data.message)
					}
				}).catch(function(error){})
			}).catch(()=>{})
		},
		closeDeviceRecycleBin(){
			var vm = this;
			egwRegisterVue.sharingSlideCancel();
		},
	},
	mounted(){
		var vm = this;
		eventBus.$off("recycle-bin-init").$on("recycle-bin-init",this.init)
	},

})

</script>
