<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<!DOCTYPE html>
<html>
<head>
	<style>
		.persent-100-flex {
			height: 100%;
			display: flex;
			flex-direction: column;
			overflow: hidden;
		}
		.persent-100-flex .el-form-item{
			margin-right: 40px;
		}
	</style>
</head>
<body>
  <div id="device_grou_add" class="persent-100-flex">
	<el-form ref="groupForm" :model="form" :rules="groupRules" label-position="top" style="flex: auto;overflow: auto;">
		<div class="group" label="<%=rb.getString("JiBenXinXi")%>">
		<el-form-item label="<%=rb.getString("SheBeiZuMingCheng")%>" prop="groupName">
			<el-input v-model="form.groupName" :disabled="readonly"></el-input>
		</el-form-item>
		<el-form-item label="<%=rb.getString("MiaoShu")%>" prop="description">
			<el-input type="textarea" v-model="form.description" :disabled="readonly"></el-input>
		</el-form-item>
		</div>

		<div class="group" label="<%=rb.getString("SheBeiXuanZe")%>">
			<el-form-item label="">
				<el-pairgrid height="220px" ref="cpairgrid" 
					:readonly="readonly"
					:query-params="groupParams"
					:title="['<%=rb.getString("SheBeiLieBiao")%>','<%=rb.getString("YiXuanSheBei")%>']" 
					@selection-change='selectChange'
					row-key="serial_number"
					:right-url="rightUrl" :left-url="leftUrl"
					:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">
					<!-- 左列表 -->
					<template slot="left">
						<el-table-column type="selection" width="45"></el-table-column>
						<el-table-column prop="connection_status" width="50">
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
						<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZuMingCheng")%>'></el-table-column>
					</template>
					<!-- 查询 toolbar -->
					<template slot="toolbar">
						<el-query type="normal" @query="queryDevice" placeholder="<%=rb.getString("XiaoZhanBianMa")%>"></el-query>
					</template>
					<!-- 结果列表 -->
					<template slot="right">
						<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
						<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZuMingCheng")%>'></el-table-column>
					</template>
				</el-pairgrid>
			</el-form-item>
		</div>
	</el-form>
	<div v-show="!readonly" style="padding: 20px 0 0 30px;">
		<el-button type="primary" @click="addDeviceGroup"><%=rb.getString("QueDing")%></el-button>
		<el-button @click="closeAddGroup"><%=rb.getString("QuXiao")%></el-button>
	</div>
  </div>
  <script type="text/javascript">
  	new Vue({
		el: '#device_grou_add',
		data() {
			var vm = this,
				validateGroupName = (rule,value,callback) => {
					if(value.trim() == vm.defaultTaskName){
						callback();
					}else if(value.trim() == '' || value.trim() == null){
						callback('<%=rb.getString("SheBeiZuMingChengTiShi")%>')
					}else{
						var reg = /^[a-zA-Z0-9_\u4e00-\u9fa5\s]{1,50}$/;
						if(reg.test(value)) {
							// 校验重名
							axios.post('${ctx}/system/deviceGroup/existGroupName.action',stringify({
								name: value,
								groupId: vm.group_id
							})).then(function(res) {
								var data = res.data;

								if(data.success == true) {
									callback('<%=rb.getString("MingChengYiCunZai")%>');
								}else {
									callback();
								}
							}).catch(function(){
								callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>'));
							});
						}else{
							callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>'))
						}
					}
				};

			return {
				form: {
					groupName: '',
					description: '',
					cellCodes: ''
				},
				groupRules:{
					groupName:[
						{validator: validateGroupName,trigger:'blur'}
					],
					description:[
						{max:100,trigger:'blur'}
					]
				},
				groupParams: {
					search_text: '',
					timeZone: timeZone
				},
				defaultTaskName: '',
				type: 'add',
				rightUrl: '',
				leftUrl: '',
				group_id: '',
				ids: ''
			}
		},
		computed: {
			readonly() {
				return this.type == 'info';
			}
		},
		methods: {
			init(params) {
				var vm = this,
					p = (params||{}),
					id = p.groupId||'',
					type = p.type;

				vm.type = type;
				vm.group_id = id;
				if(type != 'addGroup') {
					vm.type = type;
					vm.rightUrl = "${ctx}/system/device/enodeb/queryENBInfoPageList.action?isGnb=1&group_id="+id;

					//判断type 类型 区分 查看和修改 查看不允许修改
					axios.post('${ctx}/system/deviceGroup/findDeviceGroupInfo.action',stringify({
						id: id,
						timeZone: timeZone,
						isGnb: 1
					})).then(function(response){
						var data = response.data;
						vm.form.groupName = data.group_name;
						vm.form.description = data.description;
					}).catch(function(error){})
				}

				vm.leftUrl = '${ctx}/cell/cpeinfos/getEnbList.action?isGnb=1';
			},
			closeAddGroup(){
				eventBus.$emit('close-gnb-dialog');
			},
			queryDevice(text) {
				var vm = this;
				
				vm.groupParams.search_text = text;
			},
			selectChange(selection) {
				var vm = this;

				setTimeout(function(){
					var sRows = vm.$refs.cpairgrid.getData(),
						codes = sRows.map(function(item){ return item.small_cell_code;}),
						idsStr = sRows.map(function(item){ return item.small_cell_code + '_' + item.product;});
					
					vm.form.cellCodes = codes.join(',');
					
					vm.ids = idsStr.join(',');
				},50);
			},
			addDeviceGroup(){
				var param={isGnb: 1}  , vm = this , url;
			
				param.name = vm.form.groupName;
				param.desc = vm.form.description;
				param.ids = vm.ids;

				if (vm.type == "addGroup") {
					param.pid = vm.group_id;
					url = "${ctx}/system/deviceGroup/addAndAssignGroup.action";
				} else {
					url = "${ctx}/system/deviceGroup/modEnbDeviceGroup.action";
					param.group_id = vm.group_id
				}
				
				vm.$refs.groupForm.validate((valid) => {
					if(valid){
						axios.post(url,stringify(param)).then(function(response){
							let data = response.data;
							if ( data.success ){
								vm.$message.success('<%=rb.getString("ChengGong") %>');
								eventBus.$emit('refresh-group-list',gnbDeviceVue.queryGroupSearchText);
							}else {
								vm.$message.error(data.message)
							}
							vm.closeAddGroup();
							
						}).catch(function(error){})
					}else{
						return false;
					}
				})
			}
		},
		created() {
			var vm = this;

			eventBus.$off('init-group').$on('init-group',vm.init);
		}
	})
  </script>
</body>
</html>